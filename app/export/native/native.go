package native

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/anchor"
	"github.com/terra-money/core/app/export/util"
)

func ExportAllBondedLuna(app *terra.TerraApp) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	uCtx := types.UnwrapSDKContext(ctx)

	validators := app.StakingKeeper.GetAllValidators(uCtx)
	valMap := make(map[string]stakingtypes.Validator)
	for _, v := range validators {
		valMap[v.OperatorAddress] = v
	}

	snapshot := make(util.SnapshotBalanceAggregateMap)
	app.StakingKeeper.IterateUnbondingDelegations(uCtx, func(_ int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
		if anchor.AddressBLUNAHub == ubd.DelegatorAddress {
			return false
		}

		for _, entry := range ubd.Entries {
			snapshot.AppendOrAddBalance(ubd.DelegatorAddress, util.SnapshotBalance{
				Denom:   util.DenomLUNA,
				Balance: entry.Balance,
			})
		}

		return false
	})

	c := 0
	app.StakingKeeper.IterateAllDelegations(uCtx, func(del stakingtypes.Delegation) (stop bool) {
		if anchor.AddressBLUNAHub == del.DelegatorAddress {
			return false
		}

		c += 1
		if c%10000 == 0 {
			app.Logger().Info(fmt.Sprintf("Iterating delegations.. %d", c))
		}
		v, ok := valMap[del.ValidatorAddress]
		if !ok {
			return false
		}
		snapshot.AppendOrAddBalance(del.DelegatorAddress, util.SnapshotBalance{
			Denom:   util.DenomLUNA,
			Balance: v.TokensFromShares(del.Shares).TruncateInt(),
		})
		return false
	})
	return snapshot, nil
}

func ExportAllNativeBalances(app *terra.TerraApp) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	snapshot := make(util.SnapshotBalanceAggregateMap)
	app.BankKeeper.IterateAllBalances(types.UnwrapSDKContext(ctx),
		func(addr types.AccAddress, coin types.Coin) (stop bool) {
			if !coin.Amount.IsZero() && (coin.Denom == util.DenomUST || coin.Denom == util.DenomLUNA) {
				snapshot.AppendOrAddBalance(addr.String(), util.SnapshotBalance{
					Denom:   coin.Denom,
					Balance: coin.Amount,
				})
			}

			return false
		})

	return snapshot, nil
}
