package prism

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	// stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"

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
	partialPLunaHolding map[string]sdk.Int,
	partialCLunaHolding map[string]sdk.Int,
	bl *util.Blacklist,
) (map[string]map[string]sdk.Int, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	// 1. Resolve pLUNA into cLUNA. pLuna can be swapped to cLuna 1:1
	pLunaHolding, err := resolvePLunaHoldings(ctx, q, app.WasmKeeper, partialPLunaHolding, bl)
	if err != nil {
		return nil, err
	}
	cLunaHoldingInVault := pLunaHolding

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

	// 4. Merge all cLUNA holdings and dedupliate known contract holdings
	cLunaHolders = util.MergeMaps(cLunaHolders, cLunaHoldingInVault, cLunaHoldingInPair, partialCLunaHolding)
	bl.RegisterAddress(PrismCLuna, PrismVault)
	for _, b := range bl.GetAddressesByDenom(PrismCLuna) {
		if !cLunaHolders[b].IsNil() {
			delete(cLunaHolders, b)
		}
	}

	// Audit to make sure everything adds up
	// Total supply of cLuna is ~100000 different from pLuna for some reason
	cLunatotalSupply, err := util.GetCW20TotalSupply(ctx, q, PrismCLuna)
	if err != nil {
		return nil, err
	}
	err = util.AlmostEqual("cLuna supply", cLunatotalSupply, util.Sum(cLunaHolders), sdk.NewInt(200000))
	if err != nil {
		return nil, err
	}

	prismState, err := getPrismVaultState(ctx, q)
	if err != nil {
		return nil, err
	}

	err = checkUnbondingCLuna(ctx, q, prismState, cLunaHolders, make(map[string]sdk.Int))
	if err != nil {
		return nil, err
	}
	return nil, nil

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
	bl.RegisterAddress(token, pair)
	return holdingsInPool, nil
}

// Resolves pLUNA holdings in PRISM swap + pLuna ownership
// 1. For astroport LP, it should be included in partialPLunaHolding passed from the astroport export)
// 2. For edge deposits, it should also be included in partialPLunaHolding
func resolvePLunaHoldings(
	ctx context.Context,
	q wasmtypes.QueryServer,
	k wasmkeeper.Keeper,
	partialPLunaHolding map[string]sdk.Int,
	bl *util.Blacklist,
) (map[string]sdk.Int, error) {
	pLunaHoldings := make(map[string]sdk.Int)
	err := util.GetCW20AccountsAndBalances2(ctx, k, PrismPLuna, pLunaHoldings)
	if err != nil {
		return nil, err
	}

	pLunaHoldingsInPair, err := resolveCw20LpHoldings(ctx, q, k, PrismPLuna, PrismSwapCLunaPrismLP, PrismSwapPLunaPrism, bl)
	if err != nil {
		return nil, err
	}

	// Merge and deduplicate previously resolved pLuna holdings
	pLunaHoldings = util.MergeMaps(pLunaHoldings, partialPLunaHolding, pLunaHoldingsInPair)
	for _, a := range bl.GetAddressesByDenom(PrismPLuna) {
		delete(pLunaHoldings, a)
	}

	pLunaSupply, err := util.GetCW20TotalSupply(ctx, q, PrismPLuna)
	if err != nil {
		return nil, err
	}
	err = util.AlmostEqual("pLuna", pLunaSupply, util.Sum(pLunaHoldings), sdk.NewInt(200000))
	if err != nil {
		return nil, err
	}
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
	for acc, _ := range cLunaHolders {
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
