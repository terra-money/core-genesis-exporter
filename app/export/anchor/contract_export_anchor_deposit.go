package anchor

import (
	"fmt"

	"github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
)

func ExportAnchorDeposit(app *app.TerraApp, bl util.Blacklist) (util.SnapshotBalanceMap, error) {
	ctx := util.PrepCtx(app)
	logger := app.Logger()

	// scan through aUST holders, append them to accounts
	var balanceMap = make(util.BalanceMap)
	logger.Info("fetching aUST holders...")

	if err := util.GetCW20AccountsAndBalances(ctx, app.WasmKeeper, AddressAUST, balanceMap); err != nil {
		return nil, fmt.Errorf("aUST holders and balances: %v", err)
	}

	// NOTE: as per token distribution proposal, we're not converting aUST into UST
	// rather, take falt value
	// get aUST exchange rate
	//var epochStateResponse struct {
	//	ExchangeRate string `json:"exchange_rate"`
	//}
	//logger.Info("fetching aUST<>UST exchange rate...")
	//if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
	//	ContractAddress: MoneyMarketContract,
	//	QueryMsg:        getExchangeRate(height),
	//}, &epochStateResponse); err != nil {
	//	return nil, err
	//}

	// convert to SnapshotBalanceMap
	var snapshotBalanceMap = make(util.SnapshotBalanceMap)
	for addr, bal := range balanceMap {
		snapshotBalanceMap[addr] = util.SnapshotBalance{
			Denom:   util.DenomAUST,
			Balance: bal,
		}
	}

	return snapshotBalanceMap, nil
}
