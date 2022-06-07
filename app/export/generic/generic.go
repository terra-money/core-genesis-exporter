package generic

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/generic/common"
	"github.com/terra-money/core/app/export/generic/cw3"
	"github.com/terra-money/core/app/export/generic/vesting"
	"github.com/terra-money/core/app/export/util"
)

func ExportVestingContracts(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, common.ContractsMap, error) {
	ctx := util.PrepCtx(app)
	logger := app.Logger()

	// iterate through all contracts...
	contractsMap := make(common.ContractsMap)
	snapshot := make(util.SnapshotBalanceAggregateMap)

	logger.Info("Getting all contract info...")
	common.IterateAllContracts(sdk.UnwrapSDKContext(ctx), app.WasmKeeper, contractsMap)

	// handle vesting
	if vestingBalance, err := vesting.ExportVestingContracts(app, contractsMap, bl); err != nil {
		panic(err)
	} else {
		// Merge vesting balance into snapshot
		for w, sbs := range vestingBalance {
			for _, b := range sbs {
				snapshot.AppendOrAddBalance(w, b)
			}
		}
	}
	return snapshot, contractsMap, nil
}

func HandleContractBalances(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap, contractsMap common.ContractsMap, bl util.Blacklist) error {
	// handle cw3, function directly updates the snapshot
	if err := cw3.ExportCW3(app, contractsMap, snapshot, bl); err != nil {
		panic(err)
	}
	snapshot.ApplyBlackList(bl)
	return nil
}
