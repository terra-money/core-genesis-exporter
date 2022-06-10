package anchor

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	"github.com/terra-money/core/x/wasm/types"
)

// ExportAnchorDeposit iterates over aUST and count aUST balance per address
func ExportAnchorDeposit(app *app.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	logger := app.Logger()

	// scan through aUST holders, append them to accounts
	var balanceMap = make(util.BalanceMap)
	logger.Info("fetching aUST holders...")

	if err := util.GetCW20AccountsAndBalances(ctx, app.WasmKeeper, AddressAUST, balanceMap); err != nil {
		return nil, fmt.Errorf("aUST holders and balances: %v", err)
	}

	// convert to SnapshotBalanceMap
	var finalBalance = make(util.SnapshotBalanceAggregateMap)
	for addr, bal := range balanceMap {
		finalBalance.AppendOrAddBalance(addr, util.SnapshotBalance{
			Denom:   util.DenomAUST,
			Balance: bal,
		})
	}

	return finalBalance, nil
}

func GetAUstExchangeRate(app *app.TerraApp) (sdk.Dec, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	var state struct {
		ExchangeRate sdk.Dec `json:"exchange_rate"`
	}
	err := util.ContractQuery(ctx, q, &types.QueryContractStoreRequest{
		ContractAddress: MoneyMarketContract,
		QueryMsg:        []byte("{\"epoch_state\": {}}"),
	}, &state)
	if err != nil {
		return sdk.Dec{}, err
	}
	return state.ExchangeRate, nil
}
