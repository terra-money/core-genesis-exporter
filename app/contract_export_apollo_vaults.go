package app

import (
	"context"
	"encoding/json"
	"log"

	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/x/wasm/keeper"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	apolloFactory = "terra1g7jjjkt5uvkjeyhp8ecdz4e4hvtn83sud3tmh2"
)

type Strategy struct {
	Address string `json:"address"`
}

type StrategyInfo struct {
	TotalBondAmount types.Int `json:"total_bond_amount"`
	TotalShares     types.Int `json:"total_shares"`
}

type StrategyConfig struct {
	LpTokenAddr    string `json:"base_token"`
	StrategyConfig struct {
		AssetToken     string `json:"asset_token"`
		AssetTokenPair string `json:"asset_token_pair"`
	} `json:"strategy_config"`
}

type UserInfo struct {
	Shares types.Int `json:"shares"`
}

// Exports all LP ownership from Apollo vaults
// Resulting map is in the following format
// {
//   "lp_token_address_1": {
//       "wallet_address": "amount",
//   },
// }
func ExportApolloVaultLPs(app *TerraApp, q wasmtypes.QueryServer) (map[string]lpHoldings, error) {
	ctx := prepCtx(app)
	strats, err := getListOfStrategies(ctx, app.WasmKeeper)
	if err != nil {
		log.Println(err)
	}
	// log.Printf("no. of apollo strats: %d\n", len(strats))

	allLpHoldings := make(map[string]lpHoldings)
	for _, strat := range strats {
		lpHoldings, lpTokenAddr, err := getLpHoldingsForStrat(ctx, app.WasmKeeper, strat)
		if err != nil {
			panic(err)
		}
		allLpHoldings[lpTokenAddr.String()] = lpHoldings
	}
	return allLpHoldings, nil
}

func getLpHoldingsForStrat(ctx context.Context, keeper keeper.Keeper, strategyAddr sdk.AccAddress) (lpHoldings, sdk.AccAddress, error) {
	lpTokenAddr, _, err := getStrategyConfig(ctx, keeper, strategyAddr)
	if err != nil {
		return lpHoldings{}, lpTokenAddr, err
	}
	// log.Printf("vault: %s, lp token: %s, lp pair: %s\n", strategyAddr, lpTokenAddr, tokenPair)
	stratInfo, err := getStrategyInfo(ctx, keeper, strategyAddr)
	if err != nil {
		return lpHoldings{}, lpTokenAddr, err
	}
	// log.Printf("%v\n", stratInfo)
	userLpHoldings, err := getUserLpHoldings(ctx, keeper, strategyAddr, stratInfo)
	if err != nil {
		return lpHoldings{}, lpTokenAddr, err
	}
	log.Printf("len: %d", len(userLpHoldings))
	return userLpHoldings, lpTokenAddr, nil
}

func getUserLpHoldings(ctx context.Context, keeper keeper.Keeper, strategyAddr sdk.AccAddress, stratInfo StrategyInfo) (lpHoldings, error) {
	prefix := generatePrefix("user")
	lpHoldings := make(lpHoldings)
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), strategyAddr, prefix, func(key, value []byte) bool {
		// fmt.Printf("%x, %s\n", key, value)
		var userInfo UserInfo
		err := json.Unmarshal(value, &userInfo)
		if err != nil {
			panic(err)
		}
		if userInfo.Shares.IsZero() {
			return false
		}
		walletAddr := sdk.AccAddress(key)
		lpAmount := userInfo.Shares.Mul(stratInfo.TotalBondAmount).Quo(stratInfo.TotalShares)
		lpHoldings[walletAddr.String()] = lpAmount
		return false
	})
	return lpHoldings, nil
}

func getStrategyInfo(ctx context.Context, keeper keeper.Keeper, strategyAddr sdk.AccAddress) (StrategyInfo, error) {
	prefix := generatePrefix("strategy")
	var stratInfo StrategyInfo
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), strategyAddr, prefix, func(key, value []byte) bool {
		// fmt.Printf("%x, %s\n", key, value)
		err := json.Unmarshal(value, &stratInfo)
		if err != nil {
			panic(err)
		}
		return false
	})
	return stratInfo, nil
}

func getStrategyConfig(ctx context.Context, keeper keeper.Keeper, strategyAddr sdk.AccAddress) (sdk.AccAddress, sdk.AccAddress, error) {
	prefix := generatePrefix("config")
	var stratConfig StrategyConfig
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), strategyAddr, prefix, func(key, value []byte) bool {
		// fmt.Printf("%x, %s\n", key, value)
		err := json.Unmarshal(value, &stratConfig)
		if err != nil {
			panic(err)
		}
		return false
	})
	baseToken, err := AccAddressFromBase64(stratConfig.LpTokenAddr)
	if err != nil {
		panic(err)
	}
	tokenPair, err := AccAddressFromBase64(stratConfig.StrategyConfig.AssetTokenPair)
	if err != nil {
		panic(err)
	}
	return baseToken, tokenPair, nil
}

func getListOfStrategies(ctx context.Context, keeper keeper.Keeper) ([]sdk.AccAddress, error) {
	contractAddr, err := sdk.AccAddressFromBech32(apolloFactory)
	if err != nil {
		return nil, nil
	}

	prefix := generatePrefix("strategies")
	var strats []sdk.AccAddress
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), contractAddr, prefix, func(key, value []byte) bool {
		var strat Strategy
		err = json.Unmarshal(value, &strat)
		if err != nil {
			// skip if error parsing json
			return false
		}
		stratAddr, err := AccAddressFromBase64(strat.Address)
		if err != nil {
			// skip if error parsing address
			return false
		}
		strats = append(strats, stratAddr)
		return false
	})
	return strats, nil
}
