package kinetic

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/anchor"
	"github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	AddressKineticVault = "terra1w93d2h57mkhkc8wgetvnj67peakvcpzgazvf2a"
	AddressAnchorMarket = anchor.MoneyMarketContract
)

type cdp struct {
	Address string `json:"address"`
	Cdp     struct {
		TotalDeposited             sdk.Int `json:"total_deposited"`
		TotalDebt                  sdk.Int `json:"total_debt"`
		TotalCredit                sdk.Int `json:"total_credit"`
		LastAccumulatedYieldWeight sdk.Dec `json:"last_accumulated_yield_weight"`
	} `json:"cdp"`
}

// ExportKinetic don't need to care about lockdrop as it's fully unlocked
func ExportKinetic(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {

	height := app.LastBlockHeight()
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)

	bl.RegisterAddress(util.DenomAUST, AddressKineticVault)
	bl.RegisterAddress(util.DenomUST, AddressKineticVault)

	var cdps = make([]cdp, 0)

	// loop over all cdps
	var run func(string) error
	var cdpsResponse struct {
		Cdps []cdp `json:"cdps"`
	}
	run = func(startAfter string) error {
		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: AddressKineticVault,
			QueryMsg:        []byte(fmt.Sprintf("{\"cdps\":{\"start_after\":\"%s\", \"limit\":30}}", startAfter)),
		}, &cdpsResponse); err != nil {
			return fmt.Errorf("failed to query cdps: %v", err)
		}

		cdps = append(cdps, cdpsResponse.Cdps...)

		if len(cdpsResponse.Cdps) < 30 {
			return nil
		}

		return run(cdps[len(cdps)-1].Address)
	}

	if err := run(""); err != nil {
		return nil, err
	}

	// get aUST<>UST rate
	var epochStateResponse struct {
		ExchangeRate sdk.Dec `json:"exchange_rate"`
	}
	if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: AddressAnchorMarket,
		QueryMsg:        []byte(fmt.Sprintf("{\"epoch_state\":{\"block_height\":%d}}", height)),
	}, &epochStateResponse); err != nil {
		return nil, err
	}

	var finalBalance = make(util.SnapshotBalanceAggregateMap)
	for _, cdp := range cdps {
		// skip 0 deposit
		if cdp.Cdp.TotalDeposited.IsZero() {
			continue
		}

		finalBalance[cdp.Address] = []util.SnapshotBalance{
			{
				Denom:   util.DenomAUST,
				Balance: sdk.NewDecFromInt(cdp.Cdp.TotalDeposited).Quo(epochStateResponse.ExchangeRate).TruncateInt(),
			},
		}
	}

	return finalBalance, nil
}
