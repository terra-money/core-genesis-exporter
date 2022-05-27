package stader

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	wasmtypes "github.com/terra-money/core/x/wasm/types"

	"github.com/terra-money/core/app/export/util"
)

const (
	Pools     = "terra1r2vv8cyt0scyxymktyfuudqs3lgtypk72w6m3m"
	Delegator = "terra1t9ree3ftvgr70fvm6y67zsqxjms8jju8kwcsdu"
	SCC       = "terra127vwnwgwdvq94ce4ws76ddh0c699jt40dznrn2"
)

// ExportPools Export Luna holdings from the 3 staking pools.
func ExportPools(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	logger := app.Logger()
	logger.Info("Exporting Stader staking pools")

	// Pull users from user_registry map.
	// pub const USER_REGISTRY: Map<(&Addr, U64Key), UserPoolInfo> = Map::new("user_registry");
	prefix := util.GeneratePrefix("user_registry")
	delegatorAddr, err := sdk.AccAddressFromBech32(Delegator)
	if err != nil {
		return nil, err
	}

	users := []string{}
	app.WasmKeeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), delegatorAddr, prefix, func(key, value []byte) bool {
		// Filter out characters from start and end of the key.
		correctedAddress := string(key)[2:46]
		if !contains(users, correctedAddress) {
			users = append(users, correctedAddress)
		}
		return false
	})

	snapshot := make(util.SnapshotBalanceAggregateMap)
	for _, address := range users {
		totalAmount := sdk.NewInt(0)

		for i := 0; i < 3; i++ {
			var poolUserInfo struct {
				Info struct {
					Deposit *struct {
						Staked sdk.Int `json:"staked"`
					} `json:"deposit,omitempty"`
					Undelegations []struct {
						Amount sdk.Int `json:"amount"`
					} `json:"undelegations"`
				} `json:"info"`
			}

			if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
				ContractAddress: Pools,
				QueryMsg:        []byte(fmt.Sprintf("{\"get_user_computed_info\": {\"user_addr\": \"%s\", \"pool_id\": %d}}", address, i)),
			}, &poolUserInfo); err != nil {
				return nil, err
			}

			if poolUserInfo.Info.Deposit != nil {
				totalAmount = totalAmount.Add(poolUserInfo.Info.Deposit.Staked)
			}

			for _, undelegation := range poolUserInfo.Info.Undelegations {
				if !undelegation.Amount.IsZero() {
					totalAmount = totalAmount.Add(undelegation.Amount)
				}
			}
		}

		// Fetch unclaimed rewards for users.
		var sccUserInfo struct {
			User struct {
				RetainedRewards  sdk.Int `json:"retained_rewards"`
				UserStrategyInfo []struct {
					TotalRewards sdk.Int `json:"total_rewards"`
				} `json:"user_strategy_info"`
				Undelegations []struct {
					Amount sdk.Int `json:"amount"`
				} `json:"undelegation_records"`
			} `json:"user"`
		}

		if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: SCC,
			QueryMsg:        []byte(fmt.Sprintf("{\"get_user\": {\"user\": \"%s\"}}", address)),
		}, &sccUserInfo); err != nil {
			return nil, err
		}

		totalAmount = totalAmount.Add(sccUserInfo.User.RetainedRewards)

		if len(sccUserInfo.User.UserStrategyInfo) > 0 {
			totalAmount = totalAmount.Add(sccUserInfo.User.UserStrategyInfo[0].TotalRewards)
		}

		for _, undelegation := range sccUserInfo.User.Undelegations {
			totalAmount = totalAmount.Add(undelegation.Amount)
		}

		if !totalAmount.IsZero() {
			snapshot.AppendOrAddBalance(address, util.SnapshotBalance{
				Denom:   util.DenomLUNA,
				Balance: totalAmount,
			})
		}
	}

	return snapshot, nil
}
