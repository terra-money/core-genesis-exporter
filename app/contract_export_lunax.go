package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	lunaX      = "terra17y9qkl8dfkeg4py7n0g5407emqnemc3yqk5rup"
	lunaXState = "terra1xacqx447msqp46qmv8k2sq6v5jh9fdj37az898"
)

// ExportbLUNA get bLUNA balance for all accounts, multiply ER
func ExportLunaX(app *TerraApp, q wasmtypes.QueryServer) (map[string]sdk.Int, error) {
	ctx := prepCtx(app)
	logger := app.Logger()

	var balances = make(map[string]sdk.Int)
	logger.Info("fetching LunaX holders and balances...")

	if err := getCW20AccountsAndBalances(ctx, balances, bLuna, q); err != nil {
		return nil, err
	}

	// get LunaX <> Luna ER
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
