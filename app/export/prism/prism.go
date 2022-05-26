package prism

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"

	wasmkeeper "github.com/terra-money/core/x/wasm/keeper"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	PrismVault            = "terra1xw3h7jsmxvh6zse74e4099c6gl03fnmxpep76h"
	PrismCLuna            = "terra13zaagrrrxj47qjwczsczujlvnnntde7fdt0mau"
	PrismPLuna            = "terra1tlgelulz9pdkhls6uglfn5lmxarx7f2gxtdzh2"
	PrismSwapCLunaPrism   = "terra1yxgq5y6mw30xy9mmvz9mllneddy9jaxndrphvk"
	PrismSwapCLunaPrismLP = "terra1vn5c4yf70aasrq50k2xdy3vn2s8vm40wmngljh"
	PrismSwapPLunaPrism   = "terra1persuahr6f8fm6nyup0xjc7aveaur89nwgs5vs"
	PrismSwapPLunaPrismLP = "terra1rjm3ca2xh2cfm6l6nsnvs6dqzed0lgzdydy7wf"
	PrismLimitOrder       = "terra1zctyc83qmcunc5zww7hzgzmrtxhjhsj4kfgvg6"
	AstroportCLunaUST     = "terra1tkcnky57lthm2w7xce9cj5jeu9hjtq427tpwxr"
	AstrportCLunaLuna     = "terra102t6psqa45ahfd7wjskk3gcnfev32wdngkcjzd"
	AstroportPLunaLuna    = "terra1r6fchdsr8k65082u3cyrdn6x2n8hrpyrp72je0"
	PrismSwapLunaPrism    = "terra1r38qlqt69lez4nja5h56qwf4drzjpnu8gz04jd"
)

type PrismState struct {
	ExchangeRate      sdk.Dec `json:"exchange_rate"`
	LastUnbondedBatch int     `json:"last_processed_batch"`
}

func ExportContract(
	app *terra.TerraApp,
	bl *util.Blacklist,
) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Prism pLuna, cLuna holders and unbonding Luna")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	snapshot := make(util.SnapshotBalanceAggregateMap)

	// 1. Resolve pLUNA in PrismSwap and add to snapshot
	pLunaHolding, err := resolvePLunaHoldings(ctx, q, app.WasmKeeper, bl)
	if err != nil {
		return nil, err
	}
	for a, b := range pLunaHolding {
		snapshot[a] = append(snapshot[a], util.SnapshotBalance{
			Denom:   util.DenomPLUNA,
			Balance: b,
		})
	}

	// 2. Resolve cLUNA in PrismSwap - cLuna / PRISM pair
	cLunaHoldingInPair, err := resolveCw20LpHoldings(ctx, q, app.WasmKeeper, PrismCLuna, PrismSwapCLunaPrismLP, PrismSwapCLunaPrism, bl)
	if err != nil {
		return nil, err
	}

	// 3. Get all direct holders of cLUNA
	cLunaHolders := make(map[string]sdk.Int)
	err = util.GetCW20AccountsAndBalances2(ctx, app.WasmKeeper, PrismCLuna, cLunaHolders)
	if err != nil {
		return nil, err
	}

	// remove pair from cluna holders, remove here so that we can audit correctly
	delete(cLunaHolders, PrismSwapCLunaPrism)

	// 4. Merge all cLUNA holdings and dedupliate known contract holdings
	cLunaHolders = util.MergeMaps(cLunaHolders, cLunaHoldingInPair)
	bl.RegisterAddress(util.MapContractToDenom(PrismCLuna), PrismVault)

	// 5. Accumulate everything into snapshot
	for a, b := range cLunaHolders {
		snapshot[a] = append(snapshot[a], util.SnapshotBalance{
			Denom:   util.DenomCLUNA,
			Balance: b,
		})
	}

	prismState, err := getPrismVaultState(ctx, q)
	if err != nil {
		return nil, err
	}

	unbondedLunaHolding := make(map[string]sdk.Int)
	err = checkUnbondingCLuna(ctx, q, prismState, cLunaHolders, unbondedLunaHolding)
	if err != nil {
		return nil, err
	}

	for a, b := range unbondedLunaHolding {
		snapshot[a] = append(snapshot[a], util.SnapshotBalance{
			Denom:   util.DenomLUNA,
			Balance: b,
		})
	}

	bl.RegisterAddress(util.DenomLUNA, PrismVault)
	return snapshot, nil
}

func Audit(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	app.Logger().Info("Audit -- Prism")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	// check unbonding luna
	lunaInVault, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomLUNA, PrismVault)
	if err != nil {
		return err
	}
	util.AlmostEqual("unbonded luna prism", lunaInVault, snapshot.SumOfDenom(util.DenomLUNA), sdk.NewInt(100000))

	// check cluna supply
	cLunaSupply, err := util.GetCW20TotalSupply(ctx, q, PrismCLuna)
	if err != nil {
		return err
	}
	err = util.AlmostEqual("cLuna doesn't match", cLunaSupply, snapshot.SumOfDenom(util.DenomCLUNA), sdk.NewInt(200000))
	if err != nil {
		return err
	}

	// check pluna supply
	pLunaSupply, err := util.GetCW20TotalSupply(ctx, q, PrismPLuna)
	if err != nil {
		return err
	}
	err = util.AlmostEqual("pLuna doesn't match", pLunaSupply, snapshot.SumOfDenom(util.DenomPLUNA), sdk.NewInt(200000))
	if err != nil {
		return err
	}

	return nil
}

func ResolveToLuna(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap, bl util.Blacklist) error {
	app.Logger().Info("Resolving cLuna to Luna")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	snapshot.ApplyBlackList(bl)
	swapPLunaToCLuna(snapshot)
	snapshot.ApplyBlackList(bl)

	prismState, err := getPrismVaultState(ctx, q)
	if err != nil {
		return err
	}

	for _, sbs := range snapshot {
		for i, sb := range sbs {
			if sb.Denom == util.DenomCLUNA {
				sbs[i] = util.SnapshotBalance{
					Denom:   util.DenomLUNA,
					Balance: prismState.ExchangeRate.MulInt(sb.Balance).TruncateInt(),
				}
			}
		}
	}
	fmt.Println(snapshot.SumOfDenom(util.DenomLUNA))
	return nil
}

func swapPLunaToCLuna(snapshot util.SnapshotBalanceAggregateMap) {
	for _, sbs := range snapshot {
		for i, sb := range sbs {
			if sb.Denom == util.DenomPLUNA {
				sbs[i] = util.SnapshotBalance{
					Denom:   util.DenomCLUNA,
					Balance: sb.Balance,
				}
			}
		}
	}
}

func resolveCw20LpHoldings(
	ctx context.Context,
	q wasmtypes.QueryServer,
	k wasmkeeper.Keeper,
	token string,
	lp string,
	pair string,
	bl *util.Blacklist,
) (map[string]sdk.Int, error) {
	lpHoldings := make(map[string]sdk.Int)
	err := util.GetCW20AccountsAndBalances2(ctx, k, lp, lpHoldings)
	if err != nil {
		return nil, err
	}
	lpSupply, err := util.GetCW20TotalSupply(ctx, q, lp)
	if err != nil {
		return nil, err
	}
	balanceInPool, err := getBalanceInPool(ctx, q, pair, token)
	if err != nil {
		return nil, err
	}
	holdingsInPool := make(map[string]sdk.Int)
	for acc, amount := range lpHoldings {
		holdingsInPool[acc] = amount.Mul(balanceInPool).Quo(lpSupply)
	}
	bl.RegisterAddress(util.MapContractToDenom(token), pair)
	return holdingsInPool, nil
}

// Resolves pLUNA holdings in PRISM swap
func resolvePLunaHoldings(
	ctx context.Context,
	q wasmtypes.QueryServer,
	k wasmkeeper.Keeper,
	bl *util.Blacklist,
) (map[string]sdk.Int, error) {
	pLunaHoldings := make(map[string]sdk.Int)
	err := util.GetCW20AccountsAndBalances2(ctx, k, PrismPLuna, pLunaHoldings)
	if err != nil {
		return nil, err
	}
	pLunaHoldingsInPair, err := resolveCw20LpHoldings(ctx, q, k, PrismPLuna, PrismSwapPLunaPrismLP, PrismSwapPLunaPrism, bl)
	if err != nil {
		return nil, err
	}

	// Merge and deduplicate pLuna in pair contract
	pLunaHoldings = util.MergeMaps(pLunaHoldings, pLunaHoldingsInPair)
	delete(pLunaHoldings, PrismSwapPLunaPrism)

	return pLunaHoldings, nil
}

func getBalanceInPool(ctx context.Context, q wasmtypes.QueryServer, pool string, denom string) (sdk.Int, error) {
	var poolRes struct {
		Assets []struct {
			Info struct {
				Address string `json:"cw20"`
				Native  string `json:"native"`
			} `json:"info"`
			Amount sdk.Int `json:"amount"`
		} `json:"assets"`
	}
	err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: pool,
		QueryMsg:        []byte("{\"pool\":{}}"),
	}, &poolRes)
	if err != nil {
		return sdk.Int{}, err
	}
	for _, asset := range poolRes.Assets {
		if asset.Info.Address == denom || asset.Info.Native == denom {
			return asset.Amount, nil
		}
	}
	return sdk.Int{}, fmt.Errorf("denom not found in pair")
}

func getPrismVaultState(ctx context.Context, q wasmtypes.QueryServer) (PrismState, error) {
	var prismState PrismState
	err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: PrismVault,
		QueryMsg:        []byte("{\"state\":{}}"),
	}, &prismState)
	return prismState, err
}

func checkUnbondingCLuna(
	ctx context.Context,
	q wasmtypes.QueryServer,
	state PrismState,
	cLunaHolders map[string]sdk.Int,
	lunaHolders map[string]sdk.Int,
) error {
	for acc := range cLunaHolders {
		var unbonding struct {
			Requests [][]interface{} `json:"requests"`
		}
		err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: PrismVault,
			QueryMsg:        []byte(fmt.Sprintf("{\"unbond_requests\": {\"address\": \"%s\" }}", acc)),
		}, &unbonding)
		if err != nil {
			return err
		}
		for _, r := range unbonding.Requests {
			amount, ok := sdk.NewIntFromString(r[1].(string))
			if !ok {
				return fmt.Errorf("unable to parse %s to string", r[1])
			}
			// If batch is unbonding, still exists as cLuna
			if lunaHolders[acc].IsNil() {
				lunaHolders[acc] = amount
			} else {
				lunaHolders[acc] = lunaHolders[acc].Add(amount)
			}
		}
	}
	return nil
}
