package terraswap

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
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
		if lpCount%20 == 0 {
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

	app.Logger().Info("... Replace LP tokens owned by other vaults")
	for vaultAddr, vaultHoldings := range contractLpHolders {
		for lpAddr, userHoldings := range vaultHoldings {
			lpHolding, ok := lpHoldersMap[lpAddr]
			if ok {
				vaultAmount := lpHolding[vaultAddr]
				delete(lpHolding, vaultAddr)
				app.Logger().Info(fmt.Sprintf("...... Resolved for contract: %s, Added %d users", vaultAddr, len(contractLpHolders[vaultAddr][lpAddr])))
				err := util.AlmostEqual("replace astro lp", vaultAmount, util.Sum(contractLpHolders[vaultAddr][lpAddr]), sdk.NewInt(10000))
				if err != nil {
					panic(err)
				}
				for addr, amount := range userHoldings {
					if lpHolding[addr].IsNil() {
						lpHolding[addr] = sdk.ZeroInt()
					} else {
						lpHolding[addr] = lpHolding[addr].Add(amount)
					}
				}
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
