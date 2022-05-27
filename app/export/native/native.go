package native

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
)

func ExportAllBondedLuna(app *terra.TerraApp) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	uCtx := types.UnwrapSDKContext(ctx)
	var unbondingDelegations []stakingtypes.UnbondingDelegation
	app.StakingKeeper.IterateUnbondingDelegations(uCtx, func(_ int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
		unbondingDelegations = append(unbondingDelegations, ubd)
		return false
	})

	var redelegations []stakingtypes.Redelegation
	app.StakingKeeper.IterateRedelegations(uCtx, func(_ int64, red stakingtypes.Redelegation) (stop bool) {
		redelegations = append(redelegations, red)
		return false
	})

	bondedDelegations := app.StakingKeeper.GetAllDelegations(uCtx)
	validators := app.StakingKeeper.GetAllValidators(uCtx)

	valMap := make(map[string]stakingtypes.Validator)
	for _, v := range validators {
		valMap[v.OperatorAddress] = v
	}

	snapshot := make(util.SnapshotBalanceAggregateMap)
	for _, del := range bondedDelegations {
		v, ok := valMap[del.ValidatorAddress]
		if !ok {
			return nil, fmt.Errorf("validator not found %s", del.ValidatorAddress)
		}
		snapshot.AppendOrAddBalance(del.DelegatorAddress, util.SnapshotBalance{
			Denom:   util.DenomLUNA,
			Balance: v.TokensFromShares(del.Shares).TruncateInt(),
		})
	}

	for _, ub := range unbondingDelegations {
		for _, entry := range ub.Entries {
			snapshot.AppendOrAddBalance(ub.DelegatorAddress, util.SnapshotBalance{
				Denom:   util.DenomLUNA,
				Balance: entry.Balance,
			})
		}
	}
	return snapshot, nil
}

func ExportAllNativeBalances(app *terra.TerraApp) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	snapshot := make(util.SnapshotBalanceAggregateMap)
	balances := app.BankKeeper.GetAccountsBalances(types.UnwrapSDKContext(ctx))
	for _, balance := range balances {
		for _, coin := range balance.Coins {
			snapshot.AppendOrAddBalance(balance.Address, util.SnapshotBalance{
				Denom:   coin.Denom,
				Balance: coin.Amount,
			})
		}
	}
	return snapshot, nil
}
