package tfm

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	"github.com/terra-money/core/x/wasm/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

func ExportTfmFarms(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	snapshot := make(util.SnapshotBalanceAggregateMap)
	logger := app.Logger()
	qs := util.PrepWasmQueryServer(app)
	logger.Info("Exporting TFM farms")

	for _, farmAddr := range FarmContracts {
		farmAddrAccAddr, err := sdk.AccAddressFromBech32(farmAddr)
		if err != nil {
			return nil, err
		}
		prefix := util.GeneratePrefix("reward")
		app.WasmKeeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), farmAddrAccAddr, prefix, func(key, value []byte) bool {

			userAddr := string(key)
			stakerUstBal := getStakerUstBal(ctx, qs, userAddr, farmAddr)

			snapshot.AppendOrAddBalance(userAddr, util.SnapshotBalance{
				Denom:   util.DenomUST,
				Balance: stakerUstBal,
			})

			return false
		})
	}
	return snapshot, nil
}

func getStakerUstBal(ctx context.Context, q types.QueryServer, userAddr string, farmAddr string) sdk.Int {
	USTBalance := sdk.NewInt(0)

	// 1. Find the pair and LP token contract addresses.
	var initMsg struct {
		Pair    string `json:"pair"`
		LpToken string `json:"lp_token"`
	}

	if err := util.ContractInitMsg(ctx, q, &wasmtypes.QueryContractInfoRequest{
		ContractAddress: farmAddr,
	}, &initMsg); err != nil {
		panic(err)
	}

	// 2. Get total supply for LP token.
	totalSupply, err := util.GetCW20TotalSupply(ctx, q, initMsg.LpToken)
	if err != nil {
		panic(err)
	}

	// 3. Pull total balances for each side of the liquidity pool.
	var pool pool
	if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: initMsg.Pair,
		QueryMsg:        []byte("{\"pool\":{}}"),
	}, &pool); err != nil {
		panic(err)
	}

	ustInPool := sdk.NewInt(0)

	// 4. Find UST in the pair and pull it's total balance.
	if pool.Assets[0].AssetInfo.NativeToken != nil {
		if pool.Assets[0].AssetInfo.NativeToken.Denom == "uusd" {
			ustInPool = ustInPool.Add(pool.Assets[0].Amount)
		}
	} else {
		if pool.Assets[1].AssetInfo.NativeToken.Denom == "uusd" {
			ustInPool = ustInPool.Add(pool.Assets[1].Amount)
		}
	}

	// 5. Determine how many LP tokens the user had staked.
	var info stakerInfo
	if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: string(farmAddr),
		QueryMsg:        []byte(fmt.Sprintf("{\"staker_info\": {\"staker\": \"%s\"}}", userAddr))}, &info); err != nil {
		fmt.Println(err)
	}

	// 6. Calculate users share of UST in the liquidity pool.
	if !info.BondAmount.IsZero() {
		// TODO: review equation.
		USTBalance = info.BondAmount.Mul(ustInPool).Quo(totalSupply)
	}

	return USTBalance
}
