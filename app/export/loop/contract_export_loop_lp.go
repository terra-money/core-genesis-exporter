package loop

import (
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

func ExportLoopLP(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {

	factoryRun1, err := exportLoopPerFactory(app, bl, AddressLoopFactory1)
	if err != nil {
		return nil, err
	}

	factoryRun2, err := exportLoopPerFactory(app, bl, AddressLoopFactory2)
	if err != nil {
		return nil, err
	}

	return util.MergeSnapshots(factoryRun1, factoryRun2), nil
}

func exportLoopPerFactory(app *terra.TerraApp, bl util.Blacklist, factoryAddress string) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)
	keeper := app.WasmKeeper

	// iterate over pairs,
	var pools = make(poolMap)
	var pairs = make(pairMap)
	pairPrefix := util.GeneratePrefix("pair_info")
	factory, _ := sdk.AccAddressFromBech32(factoryAddress)

	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), factory, pairPrefix, func(key, value []byte) bool {
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
		if !isTargetPool(&pool) {
			return false
		}

		pools[pairAddr] = pool
		pairs[pairAddr] = pair

		return false
	})

	// for each LP token, get their token holdings
	var lpHoldersMap = make(map[string]util.BalanceMap) // lp => user => amount
	var info tokenInfo
	for _, pairInfo := range pairs {
		lpAddr := sdk.AccAddress(pairInfo.LiquidityToken).String()
		balanceMap := make(util.BalanceMap)

		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: lpAddr,
			QueryMsg:        []byte("{\"token_info\":{}}"),
		}, &info); err != nil {
			panic(fmt.Errorf("failed to query token info: %v", err))
		}

		// skip LP with no supply
		if info.TotalSupply.IsZero() {
			continue
		}

		if err := util.GetCW20AccountsAndBalances(ctx, keeper, lpAddr, balanceMap); err != nil {
			panic(fmt.Errorf("failed to iterate over LP token owners: %v", err))
		}

		lpHoldersMap[lpAddr] = balanceMap
	}

	// tackle staking here. staking => fLP => LP => user
	// fLP : LP = 1:1
	staking1, _ := sdk.AccAddressFromBech32(AddressLoopFarm1)
	staking2, _ := sdk.AccAddressFromBech32(AddressLoopFarm2)
	for lpAddr, holdermap := range lpHoldersMap {
		var flpAddrs = make([]string, 2)

		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: staking1.String(),
			QueryMsg:        []byte(fmt.Sprintf("{\"query_flp_token_from_pool_address\":{\"pool_address\":\"%s\"}}", lpAddr)),
		}, &flpAddrs[0]); err != nil {
			return nil, fmt.Errorf("error querying flp token: %v", err)
		}

		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: staking2.String(),
			QueryMsg:        []byte(fmt.Sprintf("{\"query_flp_token_from_pool_address\":{\"pool_address\":\"%s\"}}", lpAddr)),
		}, &flpAddrs[1]); err != nil {
			return nil, fmt.Errorf("error querying flp token: %v", err)
		}

		// it's always either flpAddr1 or flpAddr2, or nothing
		sort.Strings(flpAddrs)
		flpAddr := flpAddrs[1]
		if flpAddr == "" {
			continue
		}

		var lpBalance struct {
			Balance sdk.Int `json:"balance"`
		}

		for userAddr, userHolding := range holdermap {
			if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
				ContractAddress: flpAddr,
				QueryMsg:        []byte(fmt.Sprintf("{\"balance\":{\"address\":\"%s\"}}", userAddr)),
			}, &lpBalance); err != nil {
				return nil, fmt.Errorf("failed to fetch FLP balance of user: %s, flp %s", userAddr, flpAddr)
			}

			// fLP:LP = 1:1
			// add to existing balance
			holdermap[userAddr] = userHolding.Add(lpBalance.Balance)
		}
	}

	var finalBalance = make(util.SnapshotBalanceAggregateMap)
	// for each pair LP token, get their token holding, calculate their holdings per pair
	for pairAddr, pairInfo := range pairs {
		lpAddr := sdk.AccAddress(pairInfo.LiquidityToken).String()
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
