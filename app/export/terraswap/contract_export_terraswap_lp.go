package terraswap

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	wasmkeeper "github.com/terra-money/core/x/wasm/keeper"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

// ExportTerraswapLiquidity scan all factory contracts, look for pairs that have luna or ust,
// then
func ExportTerraswapLiquidity(app *terra.TerraApp, bl util.Blacklist, contractLpHolders map[string]map[string]map[string]sdk.Int) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Terraswap")
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)
	keeper := app.WasmKeeper

	// iterate over pairs,
	var pools = make(poolMap)
	var pairs = make(pairMap)
	pairPrefix := util.GeneratePrefix("pair_info")
	factory, _ := sdk.AccAddressFromBech32(AddressTerraswapFactory)
	poolCount := 0

	// Some pools are initiatialized with bad values
	// Code panics when trying to get pool details
	poolsToSkip := make(map[int]bool)
	poolsToSkip[214] = true
	poolsToSkip[215] = true

	app.Logger().Info("... Retrieving all pools")
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), factory, pairPrefix, func(key, value []byte) bool {
		poolCount += 1
		if poolsToSkip[poolCount] {
			return false
		}
		var pool pool
		var pair pair
		util.MustUnmarshalTMJSON(value, &pair)
		pairAddr := sdk.AccAddress(pair.ContractAddr).String()

		// register all pairs as blacklist.
		bl.RegisterAddress(util.DenomAUST, pairAddr)
		bl.RegisterAddress(util.DenomUST, pairAddr)
		bl.RegisterAddress(util.DenomLUNA, pairAddr)
		bl.RegisterAddress(util.DenomBLUNA, pairAddr)
		bl.RegisterAddress(util.DenomSTLUNA, pairAddr)
		bl.RegisterAddress(util.DenomPLUNA, pairAddr)
		bl.RegisterAddress(util.DenomCLUNA, pairAddr)
		bl.RegisterAddress(util.DenomSTEAK, pairAddr)
		bl.RegisterAddress(util.DenomLUNAX, pairAddr)

		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: pairAddr,
			QueryMsg:        []byte("{\"pool\":{}}"),
		}, &pool); err != nil {
			// fmt.Printf("terraswap: irregular pair, skipping: %s\n", pairAddr)
			return false
		}

		// skip non-target pools
		if !isTargetPool(&pool) || pool.Assets[0].Amount.IsZero() || pool.Assets[1].Amount.IsZero() {
			return false
		}

		pools[pairAddr] = pool
		pairs[pairAddr] = pair

		return false
	})
	app.Logger().Info(fmt.Sprintf("...... pool count: %d", len(pools)))

	app.Logger().Info("... Getting LP holders")
	lpCount := 0
	// for each LP token, get their token holdings
	var lpHoldersMap = make(map[string]util.BalanceMap) // lp => user => amount
	var info tokenInfo
	for _, pairInfo := range pairs {
		lpCount += 1
		if lpCount%100 == 0 {
			app.Logger().Info(fmt.Sprintf("...... processed %d", lpCount))
		}
		lpAddr, err := util.AccAddressFromBase64(pairInfo.LiquidityToken)
		if err != nil {
			panic(err)
		}
		balanceMap := make(util.BalanceMap)

		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: lpAddr.String(),
			QueryMsg:        []byte("{\"token_info\":{}}"),
		}, &info); err != nil {
			panic(fmt.Errorf("failed to query token info: %v", err))
		}

		// skip LP with no supply
		if info.TotalSupply.IsZero() {
			continue
		}

		if err := util.GetCW20AccountsAndBalances(ctx, keeper, lpAddr.String(), balanceMap); err != nil {
			panic(fmt.Errorf("failed to iterate over LP token owners: %v", err))
		}

		lpHoldersMap[lpAddr.String()] = balanceMap
	}

	// getAllStakingContracts(ctx, keeper, lpHoldersMap)
	app.Logger().Info("... Resolving staking ownership")
	stakingHoldings, err := getStakingHoldings(ctx, keeper)
	if err != nil {
		return nil, err
	}
	for lp, staking := range stakingHoldings {
		if lpHolding, ok := lpHoldersMap[lp]; ok {
			if amount, okk := lpHolding[staking.StakingAddr]; okk {
				app.Logger().Info(fmt.Sprintf("...... Resolving stakers: %s, Added %d users, amount %s",
					staking.StakingAddr, len(staking.Holdings), util.Sum(staking.Holdings),
				))
				err := util.AlmostEqual(
					fmt.Sprintf("terraswap staking %s lp %s\n", staking.StakingAddr, lp),
					amount,
					util.Sum(staking.Holdings),
					sdk.NewInt(1000000),
				)
				if err != nil {
					// fmt.Println(err)
					staking.Holdings = normalizeStakingHoldings(staking.Holdings, amount)
				}
				delete(lpHolding, staking.StakingAddr)
				lpHoldersMap[lp] = util.MergeMaps(lpHolding, staking.Holdings)
				util.AssertCw20Supply(ctx, qs, lp, lpHoldersMap[lp])
			}
		}
	}
	app.Logger().Info("... Replace LP tokens owned by other vaults")
	for vaultAddr, vaultHoldings := range contractLpHolders {
		for lpAddr, userHoldings := range vaultHoldings {
			lpHolding, ok := lpHoldersMap[lpAddr]
			if ok {
				vaultAmount := lpHolding[vaultAddr]
				if vaultAmount.IsNil() || vaultAmount.IsZero() {
					continue
				}
				app.Logger().Info(fmt.Sprintf("...... Resolving external vaults: %s lp %s", vaultAddr, lpAddr))
				err := util.AlmostEqual("vault amount inconsistent", vaultAmount, util.Sum(userHoldings), sdk.NewInt(5000000))
				if err != nil {
					panic(err)
				}
				for addr, amount := range userHoldings {
					if lpHolding[addr].IsNil() {
						lpHolding[addr] = amount
					} else {
						lpHolding[addr] = lpHolding[addr].Add(amount)
					}
				}
				delete(lpHolding, vaultAddr)
				util.AssertCw20Supply(ctx, qs, lpAddr, lpHolding)
			}
		}
	}

	var finalBalance = make(util.SnapshotBalanceAggregateMap)
	// for each pair LP token, get their token holding, calculate their holdings per pair
	for pairAddr, pairInfo := range pairs {
		lpAddr := pairInfo.LiquidityToken
		pool := pools[pairAddr]

		holderMap := lpHoldersMap[lpAddr]

		// iterate over LP holders, calculate how much is to be refunded
		for userAddr, lpBalance := range holderMap {

			refunds := getShareInAssets(pool, lpBalance, pool.TotalShare)
			userBalance := make([]util.SnapshotBalance, 0)

			if asset0name, ok := coalesceToBalanceDenom(pickDenomOrContractAddress(pool.Assets[0].AssetInfo)); ok {
				if !refunds[0].IsZero() {
					userBalance = append(userBalance, util.SnapshotBalance{
						Denom:   asset0name,
						Balance: refunds[0],
					})
				}

			}

			if asset1name, ok := coalesceToBalanceDenom(pickDenomOrContractAddress(pool.Assets[1].AssetInfo)); ok {
				if !refunds[1].IsZero() {
					userBalance = append(userBalance, util.SnapshotBalance{
						Denom:   asset1name,
						Balance: refunds[1],
					})
				}
			}

			// add to final balance if anything
			if len(userBalance) != 0 {
				finalBalance[userAddr] = append(finalBalance[userAddr], userBalance...)
			}
		}
	}

	return finalBalance, nil
}

type stakingInitMsg struct {
	Token                string        `json:"token"`
	LpToken              string        `json:"lp_token"`
	StakingToken         string        `json:"staking_token"`
	Pair                 string        `json:"pair"`
	DistributionSchedule []interface{} `json:"distribution_schedule"`
}

type stakingHolders struct {
	StakingAddr string
	Holdings    util.BalanceMap
}

func normalizeStakingHoldings(vaultHolding util.BalanceMap, vaultTotal sdk.Int) util.BalanceMap {
	shareTotal := util.Sum(vaultHolding)
	normalizedHolding := make(util.BalanceMap)
	for add, b := range vaultHolding {
		normalizedHolding[add] = sdk.NewDecFromInt(b).MulInt(vaultTotal).QuoInt(shareTotal).TruncateInt()
	}
	return normalizedHolding
}

func getStakingHoldings(ctx context.Context, k wasmkeeper.Keeper) (map[string]stakingHolders, error) {

	holdings := make(map[string]stakingHolders)
	for _, staking := range StakingContracts {
		stakingAddr := util.ToAddress(staking)
		var initMsg stakingInitMsg
		info, err := k.GetContractInfo(sdk.UnwrapSDKContext(ctx), stakingAddr)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(info.InitMsg, &initMsg); err != nil {
			return nil, err
		}
		var lpAddress string
		if initMsg.StakingToken != "" {
			lpAddress = initMsg.StakingToken
		} else if initMsg.LpToken != "" {
			lpAddress = initMsg.LpToken
		} else {
			continue
		}

		prefix := util.GeneratePrefix("reward")
		balances := make(map[string]sdk.Int)
		k.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), stakingAddr, prefix, func(key, value []byte) bool {
			var reward struct {
				Amount sdk.Int `json:"bond_amount"`
			}
			json.Unmarshal(value, &reward)
			holderAddr := sdk.AccAddress(key)
			balances[holderAddr.String()] = reward.Amount
			return false
		})
		holdings[lpAddress] = stakingHolders{
			StakingAddr: staking,
			Holdings:    balances,
		}
	}
	return holdings, nil
}

// This was run pre-export to get a list of staking contracts
func getAllStakingContracts(ctx context.Context, k wasmkeeper.Keeper, holders map[string]util.BalanceMap) error {
	for _, balances := range holders {
		for addr, balance := range balances {
			var initMsg stakingInitMsg
			info, err := k.GetContractInfo(sdk.UnwrapSDKContext(ctx), util.ToAddress(addr))
			if err != nil {
				continue
			}
			if err = json.Unmarshal(info.InitMsg, &initMsg); err != nil {
				continue
			}
			if len(initMsg.DistributionSchedule) > 0 {
				fmt.Printf("%s,%s,%s\n", addr, initMsg.LpToken, balance)
			}
		}
	}
	return nil
}
