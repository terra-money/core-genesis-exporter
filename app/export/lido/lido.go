package lido

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	wasmkeeper "github.com/terra-money/core/x/wasm/keeper"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

const (
	LidoHub = "terra1mtwph2juhj0rvjz7dy92gvl6xvukaxu8rfv8ts"
	StLuna  = "terra1yg3j2s986nyp5z7r2lvt0hx3r0lnd7kwvwwtsc"
	BLuna   = "terra1kc87mu460fwkqte29rquh4hc20m54fxwtsx7gp"
)

type LidoState struct {
	StLunaExchangeRate sdk.Dec `json:"stluna_exchange_rate"`
	BLunaExchangeRate  sdk.Dec `json:"bluna_exchange_rate"`
	TotalBondStLuna    sdk.Int `json:"total_bond_stluna_amount"`
	TotalBondBLuna     sdk.Int `json:"total_bond_bluna_amount"`
}

func ExportLidoContract(
	app *terra.TerraApp,
	stLunaBalances map[string]sdk.Int,
	bLunaBalances map[string]sdk.Int,
	bl *util.Blacklist,
) (map[string]util.Balance, map[string]util.Balance, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	lidoState, err := getExchangeRates(ctx, q)
	if err != nil {
		return nil, nil, err
	}

	bondedStLunaHolders := make(map[string]sdk.Int)
	err = util.GetCW20AccountsAndBalances2(ctx, app.WasmKeeper, StLuna, bondedStLunaHolders)
	if err != nil {
		return nil, nil, err
	}
	bondedBLunaHolders := make(map[string]sdk.Int)
	err = util.GetCW20AccountsAndBalances2(ctx, app.WasmKeeper, StLuna, bondedBLunaHolders)
	if err != nil {
		return nil, nil, err
	}
	unbondingBluna, unbondingStLuna, err := getUnbondingTokens(ctx, app.WasmKeeper)
	if err != nil {
		return nil, nil, err
	}
	// Merge with previously calculated from LPs and vaults etc
	stLunaBalances = util.MergeMaps(stLunaBalances, bondedStLunaHolders, unbondingStLuna)
	bLunaBalances = util.MergeMaps(bLunaBalances, bondedBLunaHolders, unbondingBluna)

	lunaBalance := util.MergeMaps(
		applyExchangeRates(stLunaBalances, lidoState.StLunaExchangeRate),
		applyExchangeRates(bLunaBalances, lidoState.BLunaExchangeRate),
	)

	stLunaTotalSupply, err := util.GetCW20TotalSupply(ctx, q, StLuna)
	if err != nil {
		return nil, nil, err
	}
	bLunaTotalSupply, err := util.GetCW20TotalSupply(ctx, q, BLuna)
	if err != nil {
		return nil, nil, err
	}

	lunaRewards, err := getLunaRewards(ctx, app.BankKeeper)
	if err != nil {
		return nil, nil, err
	}
	stLunaRewards := lunaRewards.Mul(lidoState.TotalBondStLuna.Quo(lidoState.TotalBondStLuna.Add(lidoState.TotalBondBLuna)))
	bLunaRewards := lunaRewards.Mul(lidoState.TotalBondBLuna.Quo(lidoState.TotalBondStLuna.Add(lidoState.TotalBondBLuna)))

	ustRewards, err := getUstRewards(ctx, app.BankKeeper)
	if err != nil {
		return nil, nil, err
	}
	stLunaUstRewards := ustRewards.Mul(lidoState.TotalBondStLuna.Quo(lidoState.TotalBondStLuna.Add(lidoState.TotalBondBLuna)))
	bLunaUstRewards := ustRewards.Mul(lidoState.TotalBondBLuna.Quo(lidoState.TotalBondStLuna.Add(lidoState.TotalBondBLuna)))

	for k, _ := range bondedStLunaHolders {
		lunaBalance[k] = lunaBalance[k].Add(stLunaRewards.Mul(bondedStLunaHolders[k].Quo(stLunaTotalSupply)))
	}
	for k, _ := range bondedBLunaHolders {
		lunaBalance[k] = lunaBalance[k].Add(bLunaRewards.Mul(bondedBLunaHolders[k].Quo(bLunaTotalSupply)))
	}

	ustBalance := make(map[string]util.Balance)
	for k, _ := range bondedStLunaHolders {
		ustToAdd := stLunaUstRewards.Mul(bondedStLunaHolders[k].Quo(stLunaTotalSupply))
		if ustToAdd.IsZero() {
			continue
		}
		if ustBalance[k].Balance.IsNil() {
			ustBalance[k] = util.Balance{
				Denom:   util.DenomUST,
				Balance: sdk.NewInt(0),
			}
		}
		userUstBalance := ustBalance[k]
		(&userUstBalance).AddInto(ustToAdd)
	}
	for k, _ := range bondedBLunaHolders {
		ustToAdd := bLunaUstRewards.Mul(bondedBLunaHolders[k].Quo(bLunaTotalSupply))
		if ustToAdd.IsZero() {
			continue
		}
		if ustBalance[k].Balance.IsNil() {
			ustBalance[k] = util.Balance{
				Denom:   util.DenomUST,
				Balance: sdk.NewInt(0),
			}
		}
		userUstBalance := ustBalance[k]
		(&userUstBalance).AddInto(ustToAdd)
	}

	finalLunaBalance := make(map[string]util.Balance)
	sumOfLunaBalance := sdk.NewInt(0)
	for k, b := range lunaBalance {
		finalLunaBalance[k] = util.Balance{
			Denom:   util.DenomLUNA,
			Balance: b,
		}
		sumOfLunaBalance = sumOfLunaBalance.Add(b)
	}

	fmt.Printf("%s", sumOfLunaBalance)

	// TODO: Need to verify that the total delegations + rewards = sumOfLunaBalances

	// lidoHubAddr, _ := sdk.AccAddressFromBech32(LidoHub)
	// // totalDelegations := sdk.NewInt(0)
	// // app.StakingKeeper.IterateDelegations(sdk.UnwrapSDKContext(ctx), lidoHubAddr, func(index int64, del stakingtypes.DelegationI) (stop bool) {
	// // 	totalDelegations = totalDelegations.Add(del.GetShares().TruncateInt())
	// // 	return false
	// // })

	// // unbondingDelegations := app.StakingKeeper.GetAllUnbondingDelegations(sdk.UnwrapSDKContext(ctx), lidoHubAddr)
	// // totalUnbonding := sdk.NewInt(0)
	// // for _, u := range unbondingDelegations {
	// // 	for _, e := range u.Entries {
	// // 		totalUnbonding = totalUnbonding.Add(e.Balance)
	// // 	}
	// // }

	// // fmt.Printf("difference: %s\n", sumOfLunaBalance.Sub(lunaRewards).Sub(totalDelegations))

	return finalLunaBalance, ustBalance, nil
}

func getExchangeRates(ctx context.Context, q wasmtypes.QueryServer) (LidoState, error) {
	var lidoState LidoState
	err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: LidoHub,
		QueryMsg:        []byte("{\"state\": {}}"),
	}, &lidoState)
	if err != nil {
		return LidoState{}, err
	}
	return lidoState, nil
}

func getLunaRewards(ctx context.Context, k wasmtypes.BankKeeper) (sdk.Int, error) {
	lunaRewards, err := util.GetNativeBalance(ctx, k, util.DenomLUNA, LidoHub)
	return lunaRewards, err
}

func getUstRewards(ctx context.Context, k wasmtypes.BankKeeper) (sdk.Int, error) {
	rewards, err := util.GetNativeBalance(ctx, k, util.DenomUST, LidoHub)
	return rewards, err
}

func applyExchangeRates(balances map[string]sdk.Int, exchangeRate sdk.Dec) map[string]sdk.Int {
	lunaBalances := make(map[string]sdk.Int)
	for k, v := range balances {
		lunaBalances[k] = exchangeRate.MulInt(v).RoundInt()
	}
	return lunaBalances
}

func getUnbondingTokens(ctx context.Context, k wasmkeeper.Keeper) (map[string]sdk.Int, map[string]sdk.Int, error) {
	prefix := util.GeneratePrefix("v2_wait")
	lidoHubAddr, err := sdk.AccAddressFromBech32(LidoHub)
	if err != nil {
		panic(err)
	}
	var unbondingRes struct {
		BlunaAmount  sdk.Int `json:"bluna_amount"`
		StLunaAmount sdk.Int `json:"stluna_amount"`
	}
	bLunaHolding := make(map[string]sdk.Int)
	stLunaHolding := make(map[string]sdk.Int)
	k.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), lidoHubAddr, prefix, func(key, value []byte) bool {
		// fmt.Printf("%s, %s\n", key[3:len(key)-2], value)
		// key is in the format [len("\"address\"")][address][1 byte]
		wallet := string(key[3 : len(key)-2])
		json.Unmarshal(value, &unbondingRes)
		bLunaHolding[wallet] = unbondingRes.BlunaAmount
		stLunaHolding[wallet] = unbondingRes.StLunaAmount
		return false
	})
	return bLunaHolding, stLunaHolding, nil
}
