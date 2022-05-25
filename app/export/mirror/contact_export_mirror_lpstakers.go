package mirror

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	"github.com/terra-money/core/x/wasm/keeper"
	"github.com/terra-money/core/x/wasm/types"
)

var (
	MirrorStaking = "terra17f7zu97865jmknk7p2glqvxzhduk78772ezac5"
	MirrorFactory = "terra1mzj9nsxx0lxlaxnekleqdy8xnyw2qrh3uz6h8p"
	MirAddress    = "terra15gwkyepfc6xgca5t5zefzwy42uts8l2m4g40k6"
)

// returns staking_contract_addr -> lp_token -> user -> amount
func ExportMirrorLpStakers(app *terra.TerraApp) (map[string]map[string]map[string]sdk.Int, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	assetLpMap, err := getStakingAssets(ctx, q)
	if err != nil {
		return nil, err
	}

	stakingAddr, _ := sdk.AccAddressFromBech32(MirrorStaking)
	lpHoldings, err := getLpHolders(ctx, app.WasmKeeper, stakingAddr, assetLpMap)
	if err != nil {
		return nil, err
	}

	// fmt.Println(lpHoldings)

	// assert total amounts
	for lp, users := range lpHoldings {
		contractBalance, err := util.GetCW20Balance(ctx, q, lp, MirrorStaking)
		if err != nil {
			return nil, err
		}

		// add all users balance
		usersTotalBalance := sdk.ZeroInt()
		for _, userBalance := range users {
			usersTotalBalance = usersTotalBalance.Add(userBalance)
		}

		err = util.AlmostEqual(lp, contractBalance, usersTotalBalance, sdk.NewInt(1000000))
		if err != nil {
			return nil, err
		}
	}

	// everything good, return
	return map[string]map[string]map[string]sdk.Int{
		MirrorStaking: lpHoldings,
	}, nil
}

func getLpHolders(ctx context.Context, keeper keeper.Keeper, stakingAddr sdk.AccAddress, assetLpMap map[string]string) (map[string]map[string]sdk.Int, error) {
	prefix := util.GeneratePrefix("reward")
	lpHolders := make(map[string]map[string]sdk.Int)

	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), stakingAddr, prefix, func(key, value []byte) bool {
		var reward rewardInfo
		err := json.Unmarshal(value, &reward)
		if err != nil {
			panic(err)
		}

		if reward.BondAmount.IsZero() {
			return false
		}

		// fmt.Printf("%X", key)
		walletAddr := sdk.AccAddress(key[2:22])
		stakingAsset := sdk.AccAddress(key[22:])

		// we dont count MIR/UST LP tokens, since they were accounted for in astroport generator
		if stakingAsset.String() == MirAddress {
			return false
		}

		lpTokenAddr, ok := assetLpMap[stakingAsset.String()]
		if !ok {
			// asset is delisted, skip
			return false
		}
		if lpHolders[lpTokenAddr] == nil {
			lpHolders[lpTokenAddr] = make(map[string]sdk.Int)
		}
		// fmt.Printf("address: %s balance: %s stakingAsset: %s\n", walletAddr, reward.BondAmount, stakingAsset)
		lpHolders[lpTokenAddr][walletAddr.String()] = reward.BondAmount
		return false
	})

	return lpHolders, nil
}

func getStakingAssets(ctx context.Context, q types.QueryServer) (map[string]string, error) {
	var distRes distributionInfoResponse
	assetLpMap := make(map[string]string)

	if err := util.ContractQuery(ctx, q, &types.QueryContractStoreRequest{
		ContractAddress: MirrorFactory,
		QueryMsg:        []byte("{\"distribution_info\":{}}"),
	}, &distRes); err != nil {
		return nil, err
	}

	for _, distInf := range distRes.Weights {
		if distInf[0].(string) == MirAddress {
			continue
		}

		var poolInfo poolInfoResponse
		if err := util.ContractQuery(ctx, q, &types.QueryContractStoreRequest{
			ContractAddress: MirrorStaking,
			QueryMsg:        []byte(fmt.Sprintf("{\"pool_info\":{ \"asset_token\":\"%s\"}}", distInf[0].(string))),
		}, &poolInfo); err != nil {
			return nil, err
		}

		// fmt.Printf("asset: %s lp: %s\n", distInf[0].(string), poolInfo.StakingToken)

		assetLpMap[distInf[0].(string)] = poolInfo.StakingToken
	}

	return assetLpMap, nil
}

type rewardInfo struct {
	Index         sdk.Dec `json:"index"`
	BondAmount    sdk.Int `json:"bond_amount"`
	PendingReward sdk.Int `json:"pending_reward"`
}

type poolInfoResponse struct {
	StakingToken string `json:"staking_token"`
}

type distributionInfoResponse struct {
	Weights         [][]interface{}
	LastDistributed int `json:"last_distributed"`
}
