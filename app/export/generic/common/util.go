package common

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	wasmkeeper "github.com/terra-money/core/x/wasm/keeper"
	"github.com/terra-money/core/x/wasm/types"
)

func IterateAllContracts(ctx sdk.Context, keeper wasmkeeper.Keeper, contractsInfo map[string]types.ContractInfo) {
	keeper.IterateContractInfo(ctx, func(info types.ContractInfo) bool {
		contractsInfo[info.Address] = info
		return false
	})
}
