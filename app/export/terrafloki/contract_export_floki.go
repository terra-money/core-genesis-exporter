package terrafloki

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	FlokiLLPStaking = "terra1uy5dfa2dlfnvkgm9cpvmxev5x0ky5lzlyta79w"
	FlokiPairs      = []string{
		"terra1t9ffaw69tfensrn2s0hx79tm68g7hps30unys8",
		"terra1wez50a5t3658m6zyydaeyprtwwt8gtt0dcswlw",
		"terra13wgvm70z5py4gee5r03statmxp4hjtc6we80jq",
		"terra19l0hnypxzdrp76jdyc2tjd3yexwmhz3es4uwvz",
		"terra1a5lj0yacwz2gdmsk8acwmc6hlkv23dzykmksjv",
		"terra1rjwzkud2xtltyqamkq0md0esdk3qkhjwey5xzk",
		"terra10whh69k8p9df2ccchvq03ddqqdpxlsxhte99j7",
		"terra1lq94l6w3eft9c95kj3rcex9h702ssp35qyaha3",
		"terra1k8pflcvj3mhrmthrgux2pk9a9ytthsr5trnq7z",
	}
)

// ExportTerraFloki floki pairs aren't on dexes
func ExportTerraFloki(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {

	keeper := app.WasmKeeper

	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)

	var finalBalance = make(util.SnapshotBalanceAggregateMap)

	// LLP staking
	prefix := util.GeneratePrefix("reward")
	staking, _ := sdk.AccAddressFromBech32(FlokiLLPStaking)
	llpHolderMap := make(map[string]sdk.Int) // lp balance
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), staking, prefix, func(key, value []byte) bool {
		var reward struct {
			Amount              sdk.Int `json:"bond_amount"`
			StakingTokenVersion int     `json:"staking_token_version"`
		}
		if err := json.Unmarshal(value, &reward); err != nil {
			panic(fmt.Errorf("error iterating thorugh reward keyspace: %v", err))
		}
		holderAddr := sdk.AccAddress(key)
		// Handle staking contracts that have multiple staking tokens
		if reward.StakingTokenVersion == 0 {
			llpHolderMap[holderAddr.String()] = reward.Amount
		}
		return false
	})

	for _, pairAddr := range FlokiPairs {
		var pool pool
		var pair pair

		bl.RegisterAddress(util.DenomUST, pairAddr)

		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: pairAddr,
			QueryMsg:        []byte("{\"pair\":{}}"),
		}, &pair); err != nil {
			return nil, err
		}

		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: pairAddr,
			QueryMsg:        []byte("{\"pool\": {}}"),
		}, &pool); err != nil {
			return nil, err
		}

		lpTokenAddress := pair.LiquidityToken
		var info tokenInfo
		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: lpTokenAddress,
			QueryMsg:        []byte("{\"token_info\":{}}"),
		}, &info); err != nil {
			panic(fmt.Errorf("failed to query token info: %v", err))
		}

		lpHolders := make(map[string]sdk.Int)
		if err := util.GetCW20AccountsAndBalances(ctx, keeper, lpTokenAddress, lpHolders); err != nil {
			return nil, fmt.Errorf("error iterating lp holders: %v", err)
		}

		// reset staking holding (if main FLOKI-UST pair)
		if pairAddr == FlokiPairs[0] {
			lpHolders[FlokiLLPStaking] = sdk.ZeroInt()

			// add back LLP staking holders
			for userAddr, amount := range llpHolderMap {
				if lpHolder, ok := lpHolders[userAddr]; ok {
					lpHolders[userAddr] = lpHolder.Add(amount)
				} else {
					lpHolders[userAddr] = amount
				}
			}
		}

		util.AssertCw20Supply(ctx, qs, lpTokenAddress, lpHolders)

		// iterate over LP holders, calculate how much is to be refunded
		for userAddr, lpBalance := range lpHolders {

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
				for _, bal := range userBalance {
					finalBalance.AppendOrAddBalance(userAddr, bal)
				}
			}
		}
	}

	return finalBalance, nil

}
