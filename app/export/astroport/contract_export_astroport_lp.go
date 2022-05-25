package astroport

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	AddressAstroportGenerator = "terra1zgrx9jjqrfye8swykfgmd6hpde60j0nszzupp9"
)

// ExportAstroportLP scans through all pairs on Astroport
func ExportAstroportLP(app *terra.TerraApp, bl util.Blacklist, contractLpHolders map[string]map[string]map[string]sdk.Int) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Astroport LPs")
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)
	keeper := app.WasmKeeper

	// iterate over pairs,
	var pools = make(poolMap)
	var pairs = make(pairMap)
	pairPrefix := util.GeneratePrefix("pair_info")
	factory, _ := sdk.AccAddressFromBech32(AddressAstroportFactory)

	app.Logger().Info("... Querying pair info")
	var pairAddr string
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), factory, pairPrefix, func(key, value []byte) bool {
		var pool pool
		var pair pair
		util.MustUnmarshalTMJSON(value, &pairAddr)

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
			panic(fmt.Errorf("unable to... %v", err))
		}

		// skip non-target pools
		if !isTargetPool(&pool) {
			return false
		}

		pools[pairAddr] = pool

		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: pairAddr,
			QueryMsg:        []byte("{\"pair\":{}}"),
		}, &pair); err != nil {
			panic(fmt.Errorf("unable to query pair: %v", err))
		}

		pairs[pairAddr] = pair

		return false
	})

	// for each LP token, get their token holdings
	var lpHoldersMap = make(map[string]util.BalanceMap) // lp => user => amount
	for _, pairInfo := range pairs {
		lpAddr := pairInfo.LiquidityToken
		balanceMap := make(util.BalanceMap)

		if err := util.GetCW20AccountsAndBalances(ctx, keeper, lpAddr, balanceMap); err != nil {
			panic(fmt.Errorf("failed to iterate over LP token owners: %v", err))
		}

		lpHoldersMap[lpAddr] = balanceMap
	}
	app.Logger().Info("... LPs in Generator")
	// get LP tokens in generator
	generatorPrefix := util.GeneratePrefix("user_info")
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), util.ToAddress(AddressAstroportGenerator), generatorPrefix, func(key, value []byte) bool {
		lpAddr := string(key[2:46])
		userAddress := string(key[46:90])

		// if this pool is not one of the targets, skip
		if _, isTargetLP := lpHoldersMap[lpAddr]; !isTargetLP {
			return false
		}

		var userInfo struct {
			Amount sdk.Int `json:"amount"`
		}

		if len(contractLpHolders[userAddress]) > 0 {
			for user, amount := range contractLpHolders[userAddress][lpAddr] {
				lpHoldersMap[lpAddr][user] = amount
			}
			app.Logger().Info("...... Resolved for contract: %s, Added %d users\n", userAddress, len(contractLpHolders[userAddress][lpAddr]))
			return false
		}

		util.MustUnmarshalTMJSON(value, &userInfo)

		holdInfo, userExists := lpHoldersMap[lpAddr][userAddress]
		if !userExists {
			holdInfo = sdk.ZeroInt()
		}
		lpHoldersMap[lpAddr][userAddress] = holdInfo.Add(userInfo.Amount)

		return false
	})

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
				finalBalance[userAddr] = userBalance
			}
		}
	}

	return finalBalance, nil
}
