package nebula

import (
	"fmt"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
)

var (
	AddressCommunityFund = "terra1g5py2hu8kpenqetv6xjas5z5gtaszhsuk8yn7n"
	AddressDeployer      = "terra1dtg8ypwynpk3fxac0tfkh6tcp6jata67am4k00"
)

func ExportNebulaCommunityFund(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	var finalBalance = make(util.SnapshotBalanceAggregateMap)

	bl.RegisterAddress(util.DenomUST, AddressCommunityFund)
	ctx := util.PrepCtx(app)
	uusdBalance, err := app.BankKeeper.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: AddressCommunityFund,
		Denom:   "uusd",
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching ust balance: %v", err)
	}

	finalBalance.AppendOrAddBalance(AddressDeployer, util.SnapshotBalance{
		Denom:   util.DenomUST,
		Balance: uusdBalance.Balance.Amount,
	})

	return finalBalance, nil
}