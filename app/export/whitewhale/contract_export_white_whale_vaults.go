package whitewhale

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	whiteWhaleVUST  = "terra1w0p5zre38ecdy3ez8efd5h9fvgum5s206xknrg"
	whiteWhaleVault = "terra1ec3r2esp9cqekqqvn0wd6nwrjslnwxm7fh8egy"
)

func ExportWhiteWhaleVaults(app *terra.TerraApp, q wasmtypes.QueryServer) (map[string]map[string]sdk.Int, error) {
	ctx := util.PrepCtx(app)
	vUstHoldings := make(map[string]sdk.Int)
	err := util.GetCW20AccountsAndBalances2(ctx, app.WasmKeeper, whiteWhaleVUST, vUstHoldings)
	if err != nil {
		return nil, err
	}
	fmt.Printf("no. of holders: %d\n", len(vUstHoldings))

	totalSupply, err := util.GetCW20TotalSupply(ctx, q, whiteWhaleVUST)
	if err != nil {
		return nil, err
	}
	fmt.Printf("total supply: %s\n", totalSupply)

	aUstBalance, err := util.GetCW20Balance(ctx, q, whiteWhaleVUST, whiteWhaleVault)
	if err != nil {
		return nil, err
	}
	fmt.Printf("aust balance: %s\n", aUstBalance)

	whiteWhaleVaultAddr, err := sdk.AccAddressFromBech32(whiteWhaleVault)
	if err != nil {
		return nil, err
	}

	ustBalance := app.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ctx), whiteWhaleVaultAddr, util.DenomUST).Amount

	holdings := make(map[string]map[string]sdk.Int)
	for wallet, holding := range vUstHoldings {
		holdings[util.DenomUST][wallet] = holding.Mod(ustBalance).Quo(totalSupply)
		holdings[util.AUST][wallet] = holding.Mod(aUstBalance).Quo(totalSupply)
	}
	return holdings, nil
}
