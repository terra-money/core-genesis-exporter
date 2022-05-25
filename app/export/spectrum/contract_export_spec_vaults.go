package spectrum

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	"github.com/terra-money/core/x/wasm/keeper"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	specFarms = []string{
		//SPEC-UST
		"terra17hjvrkcwn3jk2qf69s5ldxx5rjccchu35assga",
		//stLuna-LUNA
		"terra19dfth8559etgnqmnu9nwd87pjqsuufswwclcav",
		//wstETH-UST
		"terra12td8as6zhm3m9djjmpxzfue9syvrj0ewe070hf",
		//MARS-UST
		"terra1d55nmhuq75r3vf93hwkau2stts4mpe9h22herz",
		//stSOL-UST
		"terra1puxzzlcr2urp4pvx523xhq593tgpt7damnm6pc",
		//stLuna-LDO
		"terra1aeaz2w7gxu7ga8fj76mna8skhvq6ft0q0x42tv",
		//Mirror Vaults (UST)
		"terra1kehar0l76kzuvrrcwj5um72u3pjq2uvp62aruf",
		//VKR-UST
		"terra1yj34w2n24p4x7s69evjp7ukzz82ca5tvlzqa84",
		//ANC-UST
		"terra1ukm33qyqx0qcz7rupv085rgpx0tp5wzkhmcj3f",
		//Psi-UST
		"terra1jxh7hahwxlsy5cckkyhuz50a60mpn5tr0px6tq",
		//MINE-UST
		"terra1s9zqk5ksnwp8qywrmdwt2fq0a9l0zc2d2sw2an",
		//MIR-UST
		"terra1y5hd5ea9dshfwf5eysqtsey7qkdhhktmtw9y3q",
		//nLuna-Psi
		"terra19kzel57gvx42e628k6frh624x5vm2kpck9cr9c",
		//ASTRO-UST
		"terra1wn0d0zwl382pnl6hdcd8r926yx6mcqcag7v39j",
		//bLUNA-LUNA
		"terra1ejl4v53w4all7zkw8nfkw2q6d3qkpls8m4cav4",
		//LUNA-UST
		"terra1egstlx9c9pq5taja5sg0yhraa0cl5laxyvm3ln",
		//nLuna
		"terra16usjvptlpdrj7hcmy7mvdap5tttzcya7ch0can",
		//SAYVE-UST
		"terra1mr9xlwydgg0lfxvy68ylxuchzy6jdn706vwu8c",
		//ORION-UST
		"terra1p30zk5xfn34lygcyhs2us9mxwzsn88v2yqrcw6",
		//XDEFI-UST
		"terra1d9cufxz9a4px9zfzq8quqewlj24durtu6lhwfw",
		//APOLLO-UST
		"terra1zngkjhqqearpfhym9x9hnutpklduz45e9uvp9u",
	}
)

type PoolInfo struct {
	TotalLpBalance       types.Int
	LpTokenAddr          string    `json:"staking_token"`
	TotalAutoBondShares  types.Int `json:"total_auto_bond_share"`
	TotalStakeBondShares types.Int `json:"total_stake_bond_share"`
	TotalStakeBondAmount types.Int `json:"total_stake_bond_amount"`
}

type RewardInfo struct {
	RewardInfo []struct {
		TokenAddr string    `json:"asset_token"`
		LpAmount  types.Int `json:"bond_amount"`
	} `json:"reward_infos"`
}

// Exports all LP ownership from Apollo vaults
// Resulting map is in the following format
// {
//   "farm_addr": {
//     "lp_token_address_1": {
//         "wallet_address": "amount"
//     }
//   }
// }
//
// Spec Vaults may consist of more than a single pool (e.g. Mirror vault contains all Mirror Pairs)
// Logic
// 1. Get list of pools in a vault using prefix [pool_info]
// 2. Iterate through all holders using prefix [reward]
//    a. For each holder, call contract query `reward_info` to find the bond_amount.
//        i. For each pool, add the LP tokens to the resulting map
// 3. Return list of LP ownship group by LP token address and wallet address
func ExportSpecVaultLPs(app *terra.TerraApp) (map[string]map[string]map[string]sdk.Int, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	holdings := make(map[string]map[string]map[string]sdk.Int)
	for _, farmAddrStr := range specFarms {
		log.Printf("farm: %s\n", farmAddrStr)
		farmAddr, err := sdk.AccAddressFromBech32(farmAddrStr)
		if err != nil {
			return nil, err
		}

		pools, err := getSpecFarmPools(ctx, app.WasmKeeper, q, farmAddr)
		if err != nil {
			return nil, err
		}
		holding := make(map[string]map[string]sdk.Int)
		err = getSpecFarmRewards(ctx, app.WasmKeeper, q, farmAddr, pools, holding)
		holdings[farmAddrStr] = holding
		if err != nil {
			return nil, err
		}
	}
	return holdings, nil
}

func getSpecFarmPools(ctx context.Context, keeper keeper.Keeper, q wasmtypes.QueryServer, farmAddr sdk.AccAddress) (map[string]PoolInfo, error) {
	prefix := util.GeneratePrefix("pool_info")
	// var stratConfig StrategyConfig
	pools := make(map[string]PoolInfo)
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), farmAddr, prefix, func(key, value []byte) bool {
		// fmt.Printf("%x, %s\n", key, value)
		var pool PoolInfo
		err := json.Unmarshal(value, &pool)
		if err != nil {
			panic(err)
		}
		tokenAddr := sdk.AccAddress(key).String()
		lpTokenAddr, err := util.AccAddressFromBase64(pool.LpTokenAddr)
		if err != nil {
			panic(err)
		}
		pool.LpTokenAddr = lpTokenAddr.String()
		pools[tokenAddr] = pool
		return false
	})
	return pools, nil
}

func getRewardsInfo(ctx context.Context, q wasmtypes.QueryServer, farmAddr string, walletAddr string) (RewardInfo, error) {
	var reward RewardInfo
	err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: farmAddr,
		QueryMsg:        []byte(fmt.Sprintf("{\"reward_info\":{\"staker_addr\":\"%s\"}}", walletAddr)),
	}, &reward)
	if err != nil {
		return reward, err
	}

	return reward, err
}

func getSpecFarmRewards(
	ctx context.Context,
	keeper keeper.Keeper,
	q wasmtypes.QueryServer,
	farmAddr sdk.AccAddress,
	poolInfo map[string]PoolInfo,
	holdings map[string]map[string]sdk.Int,
) error {

	// Spec farm prefix format
	// [len(reward)][reward][len(wallet)][wallet][tokenAddress|denom]
	prefix := util.GeneratePrefix("reward")
	// userLpHoldings := make(map[string]lpHoldings)
	walletSeen := make(map[string]bool)
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), farmAddr, prefix, func(key, value []byte) bool {
		walletAddress := sdk.AccAddress(key[2:22])
		if walletSeen[walletAddress.String()] {
			return false
		}
		walletSeen[walletAddress.String()] = true
		rewards, err := getRewardsInfo(ctx, q, farmAddr.String(), walletAddress.String())
		if err != nil {
			panic(err)
		}
		for _, reward := range rewards.RewardInfo {
			lpAddr := poolInfo[reward.TokenAddr].LpTokenAddr
			if holdings[lpAddr] == nil {
				holdings[lpAddr] = make(map[string]sdk.Int)
			}
			holdings[lpAddr][walletAddress.String()] = reward.LpAmount
		}
		return false
	})
	fmt.Printf("number of holders: %d\n", len(walletSeen))
	return nil
}
