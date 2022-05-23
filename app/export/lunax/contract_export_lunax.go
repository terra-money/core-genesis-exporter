package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/anchor"
	util "github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	lunaX      = "terra17y9qkl8dfkeg4py7n0g5407emqnemc3yqk5rup"
	lunaXState = "terra1xacqx447msqp46qmv8k2sq6v5jh9fdj37az898"
)

// ExportbLUNA get bLUNA balance for all accounts, multiply ER
func ExportLunaX(app *terra.TerraApp, q wasmtypes.QueryServer) (map[string]sdk.Int, error) {
	ctx := util.PrepCtx(app)
	logger := app.Logger()

	var balances = make(map[string]sdk.Int)
	logger.Info("fetching LunaX holders and balances...")

	if err := util.GetCW20AccountsAndBalances(ctx, balances, anchor.BLuna, q); err != nil {
		return nil, err
	}

	// get LunaX <> Luna ER
	var bLunaHubState struct {
		ExchangeRate sdk.Dec `json:"exchange_rate"`
	}
	if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: anchor.BLunaHub,
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
