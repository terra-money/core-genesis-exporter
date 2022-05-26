package whitewhale

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
)

var (
	whiteWhaleVUST  = "terra1w0p5zre38ecdy3ez8efd5h9fvgum5s206xknrg"
	whiteWhaleVault = "terra1ec3r2esp9cqekqqvn0wd6nwrjslnwxm7fh8egy"
)

func ExportWhiteWhaleVaults(app *terra.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Whitewhale vaults")
	bl.RegisterAddress(util.DenomAUST, whiteWhaleVault)
	bl.RegisterAddress(util.DenomUST, whiteWhaleVault)
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	vUstHoldings := make(map[string]sdk.Int)
	err := util.GetCW20AccountsAndBalances2(ctx, app.WasmKeeper, whiteWhaleVUST, vUstHoldings)
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

	holdings := make(map[string]map[string]sdk.Int)
	holdings[util.DenomUST] = make(map[string]sdk.Int)
	holdings[util.DenomAUST] = make(map[string]sdk.Int)

	for wallet, holding := range vUstHoldings {
		holdings[util.DenomUST][wallet] = holding.Mul(ustBalance).Quo(totalSupply)
		holdings[util.DenomAUST][wallet] = holding.Mul(aUstBalance).Quo(totalSupply)
	}

	err = util.AlmostEqual("whitewhale ust", ustBalance, util.Sum(holdings[util.DenomUST]), sdk.NewInt(10000))
	if err != nil {
		return nil, err
	}
	err = util.AlmostEqual("whitewhale aust", aUstBalance, util.Sum(holdings[util.DenomAUST]), sdk.NewInt(10000))
	if err != nil {
		return nil, err
	}

	snapshot := make(util.SnapshotBalanceAggregateMap)
	snapshot.Add(holdings[util.DenomUST], util.DenomUST)
	snapshot.Add(holdings[util.DenomAUST], util.DenomUST)

	return snapshot, nil
}
