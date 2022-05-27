package alice

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	wasmKeeper "github.com/terra-money/core/x/wasm/keeper"
)

var (
	AliceaaUSTWrapper = "terra14el5cs9v3ezu57fc32kx2ltad4m3elg3le8twp"
)

// ExportAlice iterates over aaUST owners & extract balance
// 1aaUST = 1aUST
func ExportAlice(terra *app.TerraApp, b util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	// register blacklist
	b.RegisterAddress(util.DenomAUST, AliceaaUSTWrapper)

	// iterate through cw20 users (force iterate since alice contract doesn't implement all_accounts)
	// then get balance
	// 1aaUST = 1aUST
	ctx := util.PrepCtx(terra)
	var balances = make(util.SnapshotBalanceMap)
	if err := forceIterateAndFindWalletAndBalance(ctx, terra.WasmKeeper, AliceaaUSTWrapper, balances); err != nil {
		return nil, err
	}

	var finalBalances = make(util.SnapshotBalanceAggregateMap)
	for user, balance := range balances {
		finalBalances.AppendOrAddBalance(user, balance)
	}

	return finalBalances, nil
}

func forceIterateAndFindWalletAndBalance(ctx context.Context, keeper wasmKeeper.Keeper, aaUST string, balances map[string]util.SnapshotBalance) error {
	prefix := util.GeneratePrefix("balance")
	addr, _ := sdk.AccAddressFromBech32(aaUST)

	var bal string
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), addr, prefix, func(key, value []byte) bool {
		util.MustUnmarshalTMJSON(value, &bal)

		balInInt, _ := sdk.NewIntFromString(bal)
		balances[string(key)] = util.SnapshotBalance{
			Denom:   util.DenomAUST,
			Balance: balInInt,
		}

		return false
	})

	return nil
}
