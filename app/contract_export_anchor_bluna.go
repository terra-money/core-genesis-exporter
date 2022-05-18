package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	bLuna    = "terra1kc87mu460fwkqte29rquh4hc20m54fxwtsx7gp"
	bLunaHub = "terra1mtwph2juhj0rvjz7dy92gvl6xvukaxu8rfv8ts"
)

// ExportbLUNA get bLUNA balance for all accounts, multiply ER
func ExportbLUNA(app *TerraApp, q wasmtypes.QueryServer) (map[string]sdk.Int, error) {
	ctx := prepCtx(app)
	logger := app.Logger()

	var balances = make(map[string]sdk.Int)
	logger.Info("fetching bLUNA holders and balances...")

	if err := getCW20AccountsAndBalances(ctx, balances, bLuna, q); err != nil {
		return nil, err
	}

	// get bLUNA <> LUNA ER
	var bLunaHubState struct {
		ExchangeRate sdk.Dec `json:"exchange_rate"`
	}
	if err := contractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: bLunaHub,
		QueryMsg:        []byte("{\"state\":{}}"),
	}, &bLunaHubState); err != nil {
		return nil, err
	}

	// balance * ER
	for address, balance := range balances {
		balances[address] = bLunaHubState.ExchangeRate.MulInt(balance).TruncateInt()
	}

	return balances, nil
}
