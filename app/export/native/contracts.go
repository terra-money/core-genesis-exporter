package native

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

func GetAllContracts(app *terra.TerraApp) (map[string]bool, error) {
	ctx := util.PrepCtx(app)
	uCtx := sdk.UnwrapSDKContext(ctx)
	allContracts := make(map[string]bool)
	app.WasmKeeper.IterateContractInfo(uCtx, func(ci wasmtypes.ContractInfo) bool {
		allContracts[ci.Address] = true
		return false
	})
	return allContracts, nil
}

func SplitContractBalances(app *terra.TerraApp, contracts map[string]wasmtypes.ContractInfo, snapshot util.SnapshotBalanceAggregateMap) (user util.SnapshotBalanceAggregateMap, contract util.SnapshotBalanceAggregateMap, err error) {
	user = make(util.SnapshotBalanceAggregateMap)
	contract = make(util.SnapshotBalanceAggregateMap)

	contractAdds := make(map[string]bool)
	for addr, _ := range contracts {
		contractAdds[addr] = true
	}

	for addr, snapshot := range snapshot {
		if contractAdds[addr] {
			contract[addr] = snapshot
		} else {
			user[addr] = snapshot
		}
	}
	return user, contract, nil
}
