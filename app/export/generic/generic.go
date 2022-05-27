package generic

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/generic/common"
	"github.com/terra-money/core/app/export/generic/cw3"
	"github.com/terra-money/core/app/export/generic/vesting"
	"github.com/terra-money/core/app/export/util"
)

func ExportGenericContracts(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	logger := app.Logger()

	// iterate through all contracts...
	contractsMap := make(common.ContractsMap)

	logger.Info("[app/export/generic] getting all contract info...")
	common.IterateAllContracts(sdk.UnwrapSDKContext(ctx), app.WasmKeeper, contractsMap)

	var finalBalance = make(util.SnapshotBalanceAggregateMap)

	// handle vesting
	if vestingBalance, err := vesting.ExportVestingContracts(app, contractsMap, bl); err != nil {
		return nil, err
	} else {
		finalBalance = util.MergeSnapshots(finalBalance, vestingBalance)
	}

	// handle cw3
	if multisigBalance, err := cw3.ExportCW3(app, contractsMap, bl); err != nil {
		return nil, err
	} else {
		finalBalance = util.MergeSnapshots(finalBalance, multisigBalance)
	}

	return finalBalance, nil
}
