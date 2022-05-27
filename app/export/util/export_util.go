package util

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

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

	AddressAUST   = "terra1hzh9vpxhsk8253se0vv5jj6etdvxu3nv8z07zu"
	AddressBLUNA  = "terra1kc87mu460fwkqte29rquh4hc20m54fxwtsx7gp"
	AddressSTLUNA = "terra1yg3j2s986nyp5z7r2lvt0hx3r0lnd7kwvwwtsc"
	AddressSTEAK  = "terra1rl4zyexjphwgx6v3ytyljkkc4mrje2pyznaclv"
	AddressNLUNA  = "terra10f2mt82kjnkxqj2gepgwl637u2w4ue2z5nhz5j"
	AddressCLUNA  = "terra13zaagrrrxj47qjwczsczujlvnnntde7fdt0mau"
	AddressPLUNA  = "terra1tlgelulz9pdkhls6uglfn5lmxarx7f2gxtdzh2"
	AddressLUNAX  = "terra17y9qkl8dfkeg4py7n0g5407emqnemc3yqk5rup"

	AUST = "terra1hzh9vpxhsk8253se0vv5jj6etdvxu3nv8z07zu"
)

var contractToDenomMap map[string]string

func init() {
	contractToDenomMap = make(map[string]string)
	contractToDenomMap["terra1tlgelulz9pdkhls6uglfn5lmxarx7f2gxtdzh2"] = DenomPLUNA
	contractToDenomMap["terra17y9qkl8dfkeg4py7n0g5407emqnemc3yqk5rup"] = DenomLUNAX
	contractToDenomMap["terra13zaagrrrxj47qjwczsczujlvnnntde7fdt0mau"] = DenomCLUNA
	contractToDenomMap["terra13zaagrrrxj47qjwczsczujlvnnntde7fdt0mau"] = DenomCLUNA
	contractToDenomMap["terra1kc87mu460fwkqte29rquh4hc20m54fxwtsx7gp"] = DenomBLUNA
	contractToDenomMap["terra1yg3j2s986nyp5z7r2lvt0hx3r0lnd7kwvwwtsc"] = DenomSTLUNA
	contractToDenomMap["uluna"] = DenomLUNA
	contractToDenomMap["uusd"] = DenomUST
	contractToDenomMap[AUST] = DenomAUST
}

func MapContractToDenom(addr string) string {
	denom, ok := contractToDenomMap[addr]
	if !ok {
		panic(fmt.Errorf("contract %s not mapped to denom", addr))
	}
	return denom
}

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

var timestampsPerBlock = map[int64]time.Time{
	7544910: time.Unix(1651935577, 792),
	7790000: time.Unix(1653583088, 146),

	// test
	7684654: time.Unix(1652926192, 483),
}

func PrepCtx(app *terra.TerraApp) context.Context {
	height := app.LastBlockHeight()
	time, ok := timestampsPerBlock[height]
	if !ok {
		panic(fmt.Sprintf("Unknown target height %d", height))
	}

	ctx := app.NewContext(true, tmproto.Header{Height: height, Time: time})
	return sdktypes.WrapSDKContext(ctx)
}

func PrepWasmQueryServer(app *terra.TerraApp) wasmtypes.QueryServer {
	return wasmkeeper.NewQuerier(app.WasmKeeper)
}

func MustUnmarshalTMJSON(bz []byte, dst interface{}) {
	if err := tmjson.Unmarshal(bz, dst); err != nil {
		panic(fmt.Sprintf("unable to unmarshal; got %v. err %v", bz, err))
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

func ContractQuery(ctx context.Context, q wasmtypes.QueryServer, req *wasmtypes.QueryContractStoreRequest, res interface{}) error {
	response, err := q.ContractStore(ctx, req)
	if err != nil {
		return err
	}

	err = json.Unmarshal(response.QueryResult, res)
	if err != nil {
		return err
	}

	return nil
}

func ContractInitMsg(ctx context.Context, q wasmtypes.QueryServer, req *wasmtypes.QueryContractInfoRequest, res interface{}) error {
	response, err := q.ContractInfo(ctx, req)
	if err != nil {
		return err
	}

	unmarshalErr := json.Unmarshal(response.ContractInfo.InitMsg, res)
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

func AlmostEqual(msg string, a types.Int, b types.Int, epsilon types.Int) error {
	if a.IsNil() || b.IsNil() {
		return fmt.Errorf("inputs nil")
	}
	diff := a.Sub(b)
	var pc types.Dec
	if a.IsZero() {
		pc = sdk.NewDec(1)
	} else {
		pc = types.NewDecFromInt(diff).QuoInt(a)
	}
	if !diff.Abs().LT(epsilon) {
		return fmt.Errorf("%s difference: %s, a: %s, b: %s", msg, pc, a, b)
	}
	return nil
}

func Xor(b1 map[string]sdk.Int, b2 map[string]sdk.Int) (b3 map[string][]sdk.Int) {
	b3 = make(map[string][]sdktypes.Int)
	for k, v := range b1 {
		if b2[k].IsNil() || !b2[k].Equal(v) {
			b3[k] = append(b3[k], v)
		}
	}
	for k, v := range b2 {
		if b1[k].IsNil() || !b1[k].Equal(v) {
			if len(b3[k]) == 0 {
				b3[k] = append(b3[k], sdk.Int{})
			}
			b3[k] = append(b3[k], v)
		}
	}
	return b3
}

func AssertCw20Supply(ctx context.Context, q wasmtypes.QueryServer, cw20Addr string, holdings BalanceMap) {
	var tokenInfo struct {
		TotalSupply sdk.Int `json:"total_supply"`
	}
	ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: cw20Addr,
		QueryMsg:        []byte("{\"token_info\":{}}"),
	}, &tokenInfo)
	if err := AlmostEqual(fmt.Sprintf("token %s supply doesnt match\n", cw20Addr), tokenInfo.TotalSupply, Sum(holdings), sdk.NewInt(2000000)); err != nil {
		fmt.Println(err)
	}
}

func SaveDataToFile(file string, data interface{}) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = os.WriteFile(file, out, 0666)
	if err != nil {
		return err
	}
	return nil
}

func CachedSBA(f func(*terra.TerraApp, Blacklist) (SnapshotBalanceAggregateMap, error), file string, app *terra.TerraApp, bl Blacklist) (SnapshotBalanceAggregateMap, error) {
	if _, err := os.Stat(file); err == nil {
		data, err := os.ReadFile(file)
		if err == nil {
			var snapshot SnapshotBalanceAggregateMap
			if err = json.Unmarshal(data, &snapshot); err == nil {
				return snapshot, nil
			}
		}
	}
	snapshot, err := f(app, bl)
	if err != nil {
		return nil, err
	}
	out, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(file, out, 0666)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func CachedMap3(f func(*terra.TerraApp) (map[string]map[string]map[string]sdk.Int, error), file string, app *terra.TerraApp) (map[string]map[string]map[string]sdk.Int, error) {
	if _, err := os.Stat(file); err == nil {
		data, err := os.ReadFile(file)
		if err == nil {
			var snapshot map[string]map[string]map[string]sdk.Int
			if err = json.Unmarshal(data, &snapshot); err == nil {
				return snapshot, nil
			}
		}
	}
	snapshot, err := f(app)
	if err != nil {
		return nil, err
	}
	out, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(file, out, 0666)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}
