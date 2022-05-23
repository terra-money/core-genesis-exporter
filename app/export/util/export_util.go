package util

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/store"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	terra "github.com/terra-money/core/app"
	wasmkeeper "github.com/terra-money/core/x/wasm/keeper"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	DenomAUST = "aUST"
	DenomUST  = "uusd"
	DenomLUNA = "uluna"
	AUST      = "terra1hzh9vpxhsk8253se0vv5jj6etdvxu3nv8z07zu"
)

type allAccountsResponse struct {
	Accounts []string `json:"accounts"`
}

type balanceResponse struct {
	Balance sdktypes.Int `json:"balance"`
}

type Balance struct {
	Denom   string       `json:"denom"`
	Balance sdktypes.Int `json:"balance"`
}

type Blacklist map[string][]string

func (bl Blacklist) RegisterAddress(denom string, address string) {
	bl[denom] = append(bl[denom], address)
}

type Snapshot string

const (
	PreAttack  string = "preattack"
	PostAttack string = "preattack"
)

type lpHoldings map[string]types.Int // {wallet: amount}

func GetAllBalancesQuery(lastAccount string) json.RawMessage {
	if lastAccount == "" {
		return []byte(fmt.Sprintf("{\"all_accounts\":{\"limit\":30}}"))
	} else {
		return []byte(fmt.Sprintf("{\"all_accounts\":{\"limit\":30,\"start_after\":\"%s\"}}", lastAccount))
	}
}

func GetBalance(account string) json.RawMessage {
	return []byte(fmt.Sprintf("{\"balance\":{\"address\":\"%s\"}}", account))
}

func PrepCtx(app *terra.TerraApp) context.Context {
	height := app.LastBlockHeight()
	ctx := app.NewContext(true, tmproto.Header{Height: height})
	return sdktypes.WrapSDKContext(ctx)
}

func PrepWasmQueryServer(app *terra.TerraApp) wasmtypes.QueryServer {
	return wasmkeeper.NewQuerier(app.WasmKeeper)
}

func MustUnmarshalTMJSON(bz []byte, dst interface{}) {
	if err := tmjson.Unmarshal(bz, dst); err != nil {
		panic(fmt.Sprintf("unable to unmarshal; got %v", bz))
	}
}

func GetCW20TotalSupply(ctx context.Context, q wasmtypes.QueryServer, cw20Addr string) (sdktypes.Int, error) {
	var tokenInfo struct {
		TotalSupply sdk.Int `json:"total_supply"`
	}
	err := ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: cw20Addr,
		QueryMsg:        []byte("{\"token_info\": {} }"),
	}, &tokenInfo)
	if err != nil {
		return sdktypes.NewInt(0), err
	}
	return tokenInfo.TotalSupply, nil
}

func GetCW20Balance(ctx context.Context, q wasmtypes.QueryServer, cw20Addr string, holder string) (sdktypes.Int, error) {
	var balance struct {
		Balance sdk.Int `json:"balance"`
	}
	err := ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: cw20Addr,
		QueryMsg:        []byte(fmt.Sprintf("{\"balance\": {\"address\": \"%s\"}}", holder)),
	}, &balance)
	if err != nil {
		return sdktypes.ZeroInt(), err
	}
	return balance.Balance, nil
}

var GetCW20AccountsAndbalances = GetCW20AccountsAndBalances2

func GetCW20AccountsAndBalances2(ctx context.Context, keeper wasmkeeper.Keeper, contractAddress string, balanceMap map[string]sdktypes.Int) error {
	prefix := GeneratePrefix("balance")
	contractAddr, err := sdktypes.AccAddressFromBech32(contractAddress)
	if err != nil {
		return err
	}
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), contractAddr, prefix, func(key, value []byte) bool {
		// first and last byte is not used
		balance, ok := sdktypes.NewIntFromString(string(value[1 : len(value)-1]))
		// fmt.Printf("%s, %x, %s, %v\n", key, value, balance, ok)
		if ok {
			balanceMap[string(key)] = balance
		}
		return false
	})
	return nil
}

func GetCW20AccountsAndBalances_Inefficient(ctx context.Context, balanceMap map[string]sdktypes.Int, contractAddress string, q wasmtypes.QueryServer) error {
	var allAccounts allAccountsResponse
	var accounts []string

	var getAccounts func(lastAccount string) error
	getAccounts = func(lastAccount string) error {
		// get aUST balance
		// lcd.terra.dev/wasm/contracts/terra1..../store?query_msg={"balance":{"address":"terra1...."}}
		response, err := q.ContractStore(ctx, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: contractAddress,
			QueryMsg:        GetAllBalancesQuery(lastAccount),
		})

		if err != nil {
			return err
		}

		unmarshalErr := tmjson.Unmarshal(response.QueryResult, &allAccounts)
		if unmarshalErr != nil {
			return unmarshalErr
		}

		accounts = append(accounts, allAccounts.Accounts...)

		if len(allAccounts.Accounts) < 30 {
			return nil
		} else {
			return getAccounts(allAccounts.Accounts[len(allAccounts.Accounts)-1])
		}
	}

	if err := getAccounts(""); err != nil {
		return err
	}

	// now accounts slice is filled, get actual balances
	var balance balanceResponse
	var getAnchorUSTBalance func(account string) error
	getAnchorUSTBalance = func(account string) error {
		response, err := q.ContractStore(ctx, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: AUST,
			QueryMsg:        GetBalance(account),
		})

		if err != nil {
			return err
		}

		unmarshalErr := json.Unmarshal(response.QueryResult, &balance)
		if unmarshalErr != nil {
			return unmarshalErr
		}

		balanceMap[account] = balance.Balance

		return nil
	}

	for _, account := range accounts {
		if err := getAnchorUSTBalance(account); err != nil {
			return err
		}
	}

	return nil
}

func ContractQuery(ctx context.Context, q wasmtypes.QueryServer, req *wasmtypes.QueryContractStoreRequest, res interface{}) error {
	response, err := q.ContractStore(ctx, req)
	if err != nil {
		return err
	}

	unmarshalErr := json.Unmarshal(response.QueryResult, res)
	if unmarshalErr != nil {
		return unmarshalErr
	}

	return nil
}

// calculateIteratorStartKey calculates start key for an iterator -- useful in case where specific querier is not
// available from within the contract itself (i.e. LP stakers list from staking contract)
func calculateIteratorStartKey(store store.KVStore, ctx context.Context, q wasmtypes.QueryServer, contractAddress string, prefix []byte) ([]byte, error) {
	return nil, nil
}

func AccAddressFromBase64(s string) (sdk.AccAddress, error) {
	addr, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return sdk.AccAddress([]byte{}), err
	}
	return sdk.AccAddress(addr), nil
}

func GeneratePrefix(keys ...string) []byte {
	var prefix []byte
	for _, key := range keys {
		prefix = append(prefix, encodeLength(key)...)
		prefix = append(prefix, []byte(key)...)
	}

	return prefix
}

/// Encodes the length of a given namespace as a 2 byte big endian encoded integer
func encodeLength(key string) []byte {
	b := toByteArray(len(key))
	return []byte{b[2], b[3]}
}

func toByteArray(i int) (arr [4]byte) {
	binary.BigEndian.PutUint32(arr[0:4], uint32(i))
	return arr
}