package util

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	DenomAUST   = "aUST"
	DenomUST    = "uusd"
	DenomLUNA   = "uluna"
	DenomBLUNA  = "ubluna"
	DenomSTLUNA = "ustluna"
	DenomSTEAK  = "usteak"
	DenomNLUNA  = "unluna"
	DenomCLUNA  = "ucluna"
	DenomPLUNA  = "upluna"
	DenomLUNAX  = "ulunax"
	AUST        = "terra1hzh9vpxhsk8253se0vv5jj6etdvxu3nv8z07zu"
)

type allAccountsResponse struct {
	Accounts []string `json:"accounts"`
}

type balanceResponse struct {
	Balance sdktypes.Int `json:"balance"`
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

func GetNativeBalance(ctx context.Context, k wasmtypes.BankKeeper, denom string, account string) (sdk.Int, error) {
	accountAddr, err := sdk.AccAddressFromBech32(account)
	if err != nil {
		return sdk.NewInt(0), err
	}
	coin := k.GetBalance(sdk.UnwrapSDKContext(ctx), accountAddr, denom)
	return coin.Amount, nil
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

var GetCW20AccountsAndBalances = GetCW20AccountsAndBalances2

func GetCW20AccountsAndBalances2(ctx context.Context, keeper wasmkeeper.Keeper, contractAddress string, balanceMap map[string]sdktypes.Int) error {
	prefix := GeneratePrefix("balance")
	contractAddr, err := sdktypes.AccAddressFromBech32(contractAddress)
	if err != nil {
		return err
	}
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), contractAddr, prefix, func(key, value []byte) bool {
		// first and last byte is not used
		balance, ok := sdktypes.NewIntFromString(string(value[1 : len(value)-1]))
		// fmt.Printf("%x, %x, %s, %v\n", key, value, balance, ok)
		if ok {
			if strings.Contains(string(key), "terra") {
				balanceMap[string(key)] = balance
			} else {
				addr := sdk.AccAddress(key)
				balanceMap[addr.String()] = balance
			}
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

func MergeMaps(m0 map[string]sdk.Int, ms ...map[string]sdk.Int) map[string]sdk.Int {
	newMap := make(map[string]sdk.Int)
	for k, v := range m0 {
		newMap[k] = v
	}
	for _, m := range ms {
		for k, v := range m {
			if newMap[k].IsNil() {
				newMap[k] = sdk.NewInt(0)
			}
			newMap[k] = newMap[k].Add(v)
		}

	}
	return newMap
}

func Sum(m map[string]sdk.Int) sdk.Int {
	sum := sdk.NewInt(0)
	for _, v := range m {
		if !v.IsNil() {
			sum = sum.Add(v)
		}
	}
	return sum
}

func ToCsv(filePath string, headers []string, data [][]string) {
	f, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	_, err = f.Write([]byte(fmt.Sprintf("%s\n", strings.Join(headers, ","))))
	if err != nil {
		panic(err)
	}

	for _, r := range data {
		_, err = f.Write([]byte(fmt.Sprintf("%s\n", strings.Join(r, ","))))
	}
}

func ToAddress(addr string) sdk.AccAddress {
	if acc, err := sdk.AccAddressFromBech32(addr); err != nil {
		panic(fmt.Errorf("cannot convert addres %s to sdk.AccAddress", addr))
	} else {
		return acc
	}
}
