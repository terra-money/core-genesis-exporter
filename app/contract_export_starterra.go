package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	starTerraAddr = "terra1yzewp648fwq7ymlfdg5h90dfzk5y2hf6kk9pdm"
	startAfter    = "terra1a0uw7tlk03lc8ns5adnhgxvnllwdxgalv2fj6h"
)

// ExportbLUNA get bLUNA balance for all accounts, multiply ER
func ExportStarTerra(app *TerraApp, q wasmtypes.QueryServer) (map[string]sdk.Int, error) {
	ctx := prepCtx(app)
	logger := app.Logger()

	var balances = make(map[string]sdk.Int)
	logger.Info("fetching Starterra holders and balances...")

	if err := getCW20AccountsAndBalances(ctx, balances, starTerraAddr, q); err != nil {
		return nil, err
	}

	// get LunaX <> Luna ER
	var starTerraAccount struct {
		Users struct {
			Funder         string `json:"funders"`
			AvailableFunds string `json:"available_funds"`
			SpentFunds     string `json:"spent_funds"`
		} `json: "users"`
	}

	if err := contractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: starTerraAddr,
		QueryMsg:        []byte("{\"funders\":{\"limit\": 1024,\"start_after\": terra1a0uw7tlk03lc8ns5adnhgxvnllwdxgalv2fj6h}}"),
	}, &starTerraAccount); err != nil {
		return nil, err
	}

	return balances, nil
}
