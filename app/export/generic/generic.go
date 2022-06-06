package generic

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/generic/common"
	"github.com/terra-money/core/app/export/generic/cw3"
	"github.com/terra-money/core/app/export/generic/vesting"
	"github.com/terra-money/core/app/export/util"
)

func ExportGenericContracts(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap, bl util.Blacklist) (common.ContractsMap, error) {
	ctx := util.PrepCtx(app)
	logger := app.Logger()

	// iterate through all contracts...
	contractsMap := make(common.ContractsMap)

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

	// handle cw3, function directly updates the snapshot
	if err := cw3.ExportCW3(app, contractsMap, snapshot, bl); err != nil {
		panic(err)
	}

	return contractsMap, nil
}
