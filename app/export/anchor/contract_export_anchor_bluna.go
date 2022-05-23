package anchor

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var ()

// ExportbLUNA get bLUNA balance for all accounts, multiply ER
func ExportbLUNA(terra *app.TerraApp, q wasmtypes.QueryServer) (map[string]sdk.Int, error) {
	ctx := util.PrepCtx(terra)
	logger := terra.Logger()

	var balances = make(map[string]sdk.Int)
	logger.Info("fetching bLUNA holders and balances...")

	if err := util.GetCW20AccountsAndBalances(ctx, balances, BLuna, q); err != nil {
		return nil, err
	}

	// get bLUNA <> LUNA ER
	var bLunaHubState struct {
		ExchangeRate sdk.Dec `json:"exchange_rate"`
	}
	if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: BLunaHub,
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
