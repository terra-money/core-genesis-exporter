package stader

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	LunaX      = "terra17y9qkl8dfkeg4py7n0g5407emqnemc3yqk5rup"
	LunaXState = "terra1xacqx447msqp46qmv8k2sq6v5jh9fdj37az898"
)

// ExportLunaX get LunaX balance for all accounts, multiply ER
func ExportLunaX(app *terra.TerraApp) (util.SnapshotBalanceMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	balances := make(util.SnapshotBalanceMap)

	logger := app.Logger()
	logger.Info("fetching LunaX holders and balances...")

	var lunaxBalances = make(util.BalanceMap)

	if err := util.GetCW20AccountsAndBalances(ctx, app.WasmKeeper, LunaX, balances); err != nil {
		return nil, err
	}

	exchangeRate, err := GetLunaXExchangeRate(ctx, q)
	if err != nil {
		return nil, err
	}

	// balance * ER
	for address, balance := range lunaxBalances {
		balances[address] = util.SnapshotBalance{
			Denom:   util.DenomLUNA,
			Balance: exchangeRate.MulInt(balance).TruncateInt(),
		}
	}

	return balances, nil
}

// GetLunaXExchangeRate Get the exchange rate from LunaX to Luna.
func GetLunaXExchangeRate(ctx context.Context, q wasmtypes.QueryServer) (sdk.Dec, error) {
	// get LunaX <> Luna ER
	var lunaxStateResponse struct {
		State struct {
			ExchangeRate sdk.Dec `json:"exchange_rate"`
		} `json:"state"`
	}

	if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: LunaXState,
		QueryMsg:        []byte("{\"state\":{}}"),
	}, &lunaxStateResponse); err != nil {
		return sdk.NewDec(0), err
	}

	return lunaxStateResponse.State.ExchangeRate, nil
}
