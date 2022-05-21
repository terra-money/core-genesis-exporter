package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	whiteWhaleVUST  = "terra1w0p5zre38ecdy3ez8efd5h9fvgum5s206xknrg"
	whiteWhaleVault = "terra1ec3r2esp9cqekqqvn0wd6nwrjslnwxm7fh8egy"
)

func ExportWhiteWhaleVaults(app *TerraApp, q wasmtypes.QueryServer) (map[string]map[string]sdk.Int, error) {
	ctx := prepCtx(app)
	vUstHoldings := make(map[string]sdk.Int)
	err := getCW20AccountsAndBalances2(ctx, app.WasmKeeper, whiteWhaleVUST, vUstHoldings)
	if err != nil {
		return nil, err
	}
	fmt.Printf("no. of holders: %d\n", len(vUstHoldings))

	totalSupply, err := getCW20TotalSupply(ctx, q, whiteWhaleVUST)
	if err != nil {
		return nil, err
	}
	fmt.Printf("total supply: %s\n", totalSupply)

	aUstBalance, err := getCW20Balance(ctx, q, whiteWhaleVUST, whiteWhaleVault)
	if err != nil {
		return nil, err
	}
	fmt.Printf("aust balance: %s\n", aUstBalance)

	whiteWhaleVaultAddr, err := sdk.AccAddressFromBech32(whiteWhaleVault)
	if err != nil {
		return nil, err
	}

	ustBalance := app.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ctx), whiteWhaleVaultAddr, "uusd").Amount

	holdings := make(map[string]map[string]sdk.Int)
	for wallet, holding := range vUstHoldings {
		holdings["uust"][wallet] = holding.Mod(ustBalance).Quo(totalSupply)
		holdings[aUST][wallet] = holding.Mod(aUstBalance).Quo(totalSupply)
	}
	return holdings, nil
}
