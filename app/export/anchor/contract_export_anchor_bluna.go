package anchor

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

// ExportbLUNA get bLUNA balance for all accounts, multiply ER
func ExportbLUNA(app *app.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceMap, error) {
	bl.RegisterAddress(util.DenomLUNA, AddressBLUNAHub)

	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	logger := app.Logger()

	var balanceMap = make(util.BalanceMap)
	logger.Info("fetching bLUNA holders and balances...")

	if err := util.GetCW20AccountsAndBalances(ctx, app.WasmKeeper, AddressBLUNAToken, balanceMap); err != nil {
		return nil, err
	}

	// get bLUNA <> LUNA ER
	var bLunaHubState struct {
		ExchangeRate sdk.Dec `json:"exchange_rate"`
	}
	if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: AddressBLUNAHub,
		QueryMsg:        []byte("{\"state\":{}}"),
	}, &bLunaHubState); err != nil {
		return nil, err
	}

	// balance * ER
	var finalBalance = make(util.SnapshotBalanceMap)
	for address, balance := range balanceMap {
		finalBalance[address] = util.SnapshotBalance{
			Denom:   util.DenomLUNA,
			Balance: bLunaHubState.ExchangeRate.MulInt(balance).TruncateInt(),
		}
	}

	return finalBalance, nil
}
