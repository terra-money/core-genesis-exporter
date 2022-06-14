package whitewhale

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	whiteWhaleVUST        = "terra1w0p5zre38ecdy3ez8efd5h9fvgum5s206xknrg"
	whiteWhaleVault       = "terra1ec3r2esp9cqekqqvn0wd6nwrjslnwxm7fh8egy"
	whitewhaleTreasury    = "terra1cnt2dls25u40wqyjgq72stuyjrwn0u5r6m5sm5"
	whitewhaleVUstUstLp   = "terra16w76qmlwdevxvt9xnfafmczrjyar6rh5rtsyhw"
	whitewhaleVUstWhaleLp = "terra12arl49w7t4xpq7krtv43t3dg6g8kn2xxyaav56"
)

type pool struct {
	Assets     [2]asset `json:"assets"`
	TotalShare sdk.Int  `json:"total_share"`
}

type pair struct {
	AssetInfos     [2]assetInfo `json:"asset_infos"`
	ContractAddr   []byte       `json:"contract_addr"`
	LiquidityToken string       `json:"liquidity_token"`
}

type assetInfo struct {
	Token *struct {
		ContractAddr string `json:"contract_addr"`
	} `json:"token,omitempty"`
	NativeToken *struct {
		Denom string `json:"denom"`
	} `json:"native_token,omitempty"`
}

type asset struct {
	AssetInfo assetInfo `json:"info"`
	Amount    sdk.Int   `json:"amount"`
}

func ExportWhiteWhaleVaults(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Whitewhale vaults")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	vUstHoldings := make(map[string]sdk.Int)
	err := util.GetCW20AccountsAndBalances(ctx, app.WasmKeeper, whiteWhaleVUST, vUstHoldings)
	if err != nil {
		return nil, err
	}

	totalSupply, err := util.GetCW20TotalSupply(ctx, q, whiteWhaleVUST)
	if err != nil {
		return nil, err
	}

	aUstBalance, err := util.GetCW20Balance(ctx, q, util.AUST, whiteWhaleVault)
	if err != nil {
		return nil, err
	}

	ustBalance, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomUST, whiteWhaleVault)
	if err != nil {
		return nil, err
	}
	resolveVUstLps(app, whitewhaleVUstUstLp, vUstHoldings)
	resolveVUstLps(app, whitewhaleVUstWhaleLp, vUstHoldings)

	holdings := make(map[string]map[string]sdk.Int)
	holdings[util.DenomUST] = make(map[string]sdk.Int)
	holdings[util.DenomAUST] = make(map[string]sdk.Int)

	for wallet, holding := range vUstHoldings {
		holdings[util.DenomUST][wallet] = holding.Mul(ustBalance).Quo(totalSupply)
		holdings[util.DenomAUST][wallet] = holding.Mul(aUstBalance).Quo(totalSupply)
	}

	snapshot := make(util.SnapshotBalanceAggregateMap)
	snapshot.Add(holdings[util.DenomUST], util.DenomUST)
	snapshot.Add(holdings[util.DenomAUST], util.DenomAUST)
	bl.RegisterAddress(util.DenomAUST, whiteWhaleVault)
	bl.RegisterAddress(util.DenomUST, whiteWhaleVault)

	// Handle treasury since it is not a cw3
	ci, err := app.WasmKeeper.GetContractInfo(sdk.UnwrapSDKContext(ctx), util.ToAddress(whiteWhaleVault))
	if err != nil {
		return nil, err
	}

	snapshot[ci.Admin] = snapshot[whitewhaleTreasury]
	delete(snapshot, whitewhaleTreasury)
	return snapshot, nil
}

func Audit(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	aUstBalance, err := util.GetCW20Balance(ctx, q, util.AUST, whiteWhaleVault)
	if err != nil {
		return err
	}
	ustBalance, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomUST, whiteWhaleVault)
	if err != nil {
		return err
	}

	err = util.AlmostEqual("whitewhale ust", ustBalance, snapshot.SumOfDenom(util.DenomUST), sdk.NewInt(10000))
	if err != nil {
		return err
	}
	err = util.AlmostEqual("whitewhale aust", aUstBalance, snapshot.SumOfDenom(util.DenomAUST), sdk.NewInt(10000))
	if err != nil {
		return err
	}

	ci, err := app.WasmKeeper.GetContractInfo(sdk.UnwrapSDKContext(ctx), util.ToAddress(whiteWhaleVault))
	if err != nil {
		return err
	}

	if len(snapshot[whitewhaleTreasury]) > 0 || len(snapshot[ci.Admin]) == 0 {
		return fmt.Errorf("whitewhale treasury error")
	}
	return nil
}

func resolveVUstLps(app *terra.TerraApp, lpAddress string, vUstHolding map[string]sdk.Int) error {

	lpVustAllocation := vUstHolding[lpAddress]
	if lpVustAllocation.IsNil() {
		return fmt.Errorf("no vUST allocated to LP: %s", lpAddress)
	}

	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)
	var pool pool
	if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{ContractAddress: lpAddress,
		QueryMsg: []byte("{\"pool\":{}}"),
	}, &pool); err != nil {
		return err
	}

	var pair pair
	if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: lpAddress,
		QueryMsg:        []byte("{\"pair\":{}}"),
	}, &pair); err != nil {
		return err
	}

	// Find LP token ownership
	lpHoldings := make(map[string]sdk.Int)
	err := util.GetCW20AccountsAndBalances(ctx, app.WasmKeeper, pair.LiquidityToken, lpHoldings)
	if err != nil {
		return err
	}

	// Split and assign vUST to LP holders
	var vUstReserve sdk.Int
	if pool.Assets[0].AssetInfo.Token != nil && pool.Assets[0].AssetInfo.Token.ContractAddr == whiteWhaleVUST {
		vUstReserve = pool.Assets[0].Amount
	}
	if pool.Assets[1].AssetInfo.Token != nil && pool.Assets[1].AssetInfo.Token.ContractAddr == whiteWhaleVUST {
		vUstReserve = pool.Assets[1].Amount
	}
	if vUstReserve.IsNil() {
		return fmt.Errorf("vUST pair not found in lp: %s", lpAddress)
	}
	totalSupply := pool.TotalShare

	for addr, lpAmount := range lpHoldings {
		if vUstHolding[addr].IsNil() {
			vUstHolding[addr] = sdk.NewInt(0)
		}
		vUstHolding[addr] = vUstHolding[addr].Add(lpAmount.Mul(vUstReserve).Quo(totalSupply))
	}
	delete(vUstHolding, lpAddress)
	return nil
}
