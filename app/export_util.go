package app

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/store"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	wasmkeeper "github.com/terra-money/core/x/wasm/keeper"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

type allAccountsResponse struct {
	Accounts []string `json:"accounts"`
}

type balanceResponse struct {
	Balance sdktypes.Int `json:"balance"`
}

type balance struct {
	Denom   string       `json:"denom"`
	Balance sdktypes.Int `json:"balance"`
}

type lpHoldings map[string]types.Int // {wallet: amount}

func prepCtx(app *TerraApp) context.Context {
	height := app.LastBlockHeight()
	ctx := app.NewContext(true, tmproto.Header{Height: height})
	return sdktypes.WrapSDKContext(ctx)
}

func getCW20AccountsAndBalances2(ctx context.Context, keeper wasmkeeper.Keeper, contractAddress string, balanceMap map[string]sdktypes.Int) error {
	prefix := generatePrefix("balance")
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

func getCW20AccountsAndBalances(ctx context.Context, balanceMap map[string]sdktypes.Int, contractAddress string, q wasmtypes.QueryServer) error {
	var allAccounts allAccountsResponse
	var accounts []string

	var getAccounts func(lastAccount string) error
	getAccounts = func(lastAccount string) error {
		// get aUST balance
		// lcd.terra.dev/wasm/contracts/terra1..../store?query_msg={"balance":{"address":"terra1...."}}
		response, err := q.ContractStore(ctx, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: contractAddress,
			QueryMsg:        getAllBalancesQuery(lastAccount),
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
			ContractAddress: aUST,
			QueryMsg:        getBalance(account),
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

func contractQuery(ctx context.Context, q wasmtypes.QueryServer, req *wasmtypes.QueryContractStoreRequest, res interface{}) error {
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

func generatePrefix(keys ...string) []byte {
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
