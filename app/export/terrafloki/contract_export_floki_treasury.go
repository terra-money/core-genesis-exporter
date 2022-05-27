package terrafloki

import (
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
)

var (
	AddressRefund   = "terra18g85ql9y9dendkp9m2qlea9ve3cddsjydldugv"
	AddressRefundTo = "terra14qjm8ds0adqx9rsuxg97vm3av4y2uvdk7key4l"
	AddressMarket   = "terra1mut6ymgk46cnvddl8ut7qveyy5s57lgxrer9uj"
	AddressMarketTo = "terra1y3wh6amdh7wets54f0jzz6fy3skwyum55yr8hs"
)

// ExportFlokiRefunds only ever held UST/aUST
func ExportFlokiRefunds(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)

	bl.RegisterAddress(util.DenomUST, AddressRefund)
	bl.RegisterAddress(util.DenomAUST, AddressRefund)
	bl.RegisterAddress(util.DenomUST, AddressMarket)
	bl.RegisterAddress(util.DenomAUST, AddressMarket)

	//
	var finalBalance = make(util.SnapshotBalanceAggregateMap)

	// map funds from a -> b
	{
		uusdBalance, err := app.BankKeeper.Balance(ctx, &banktypes.QueryBalanceRequest{
			Address: AddressRefund,
			Denom:   "uusd",
		})
		if err != nil {
			return nil, err
		}

		austBalance, err := util.GetCW20Balance(ctx, qs, util.AddressAUST, AddressRefund)

		finalBalance.AppendOrAddBalance(AddressRefundTo, util.SnapshotBalance{
			Denom:   util.DenomUST,
			Balance: uusdBalance.Balance.Amount,
		})
		finalBalance.AppendOrAddBalance(AddressRefundTo, util.SnapshotBalance{
			Denom:   util.DenomAUST,
			Balance: austBalance,
		})
	}

	{
		uusdBalance, err := app.BankKeeper.Balance(ctx, &banktypes.QueryBalanceRequest{
			Address: AddressMarket,
			Denom:   "uusd",
		})
		if err != nil {
			return nil, err
		}

		austBalance, err := util.GetCW20Balance(ctx, qs, util.AddressAUST, AddressMarket)

		finalBalance.AppendOrAddBalance(AddressMarketTo, util.SnapshotBalance{
			Denom:   util.DenomUST,
			Balance: uusdBalance.Balance.Amount,
		})
		finalBalance.AppendOrAddBalance(AddressMarketTo, util.SnapshotBalance{
			Denom:   util.DenomAUST,
			Balance: austBalance,
		})
	}

	return finalBalance, nil
}