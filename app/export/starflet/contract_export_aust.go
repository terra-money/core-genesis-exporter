package starflet

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
)

const (
	Arbitrage = "terra1ndvfjs47eax9yxkc5tge2awlahswry3tg76zvj"
	vSWAP     = "terra1tgsex5ncsutdpl202mhgmldwtcp2w8nyg6a5gq"
)

// ExportArbitrageAUST Export locked funds in Arbitrage Contract.
func ExportArbitrageAUST(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	logger := app.Logger()

	// balance map for vSWAP holders
	var balanceMap = make(util.BalanceMap)

	totalSupply, err := util.GetCW20TotalSupply(ctx, q, vSWAP)
	if err != nil {
		return nil, err
	}

	logger.Info("fetching starflet aUST holders...")
	if err := util.GetCW20AccountsAndBalances(ctx, app.WasmKeeper, vSWAP, balanceMap); err != nil {
		return nil, err
	}

	aUSTBalance, err := util.GetCW20Balance(ctx, q, util.AUST, Arbitrage)
	if err != nil {
		return nil, err
	}

	var snapshotBalance = make(util.SnapshotBalanceAggregateMap)
	for addr, balance := range balanceMap {
		ratio := sdk.NewDecFromInt(balance).QuoInt(totalSupply)
		aUSTBalance := ratio.MulInt(aUSTBalance).TruncateInt()
		if aUSTBalance.IsZero() {
			continue
		}

		snapshotBalance.AppendOrAddBalance(addr, util.SnapshotBalance{
			Denom:   util.DenomAUST,
			Balance: aUSTBalance,
		})
	}

	bl.RegisterAddress(util.DenomAUST, Arbitrage)
	return snapshotBalance, nil
}
