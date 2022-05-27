package lido

import (
	"context"
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	// stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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

func ExportBSTLunaHolders(
	app *terra.TerraApp,
	snapshot util.SnapshotBalanceAggregateMap,
	bl util.Blacklist,
) error {
	app.Logger().Info("Exporting bLUNA and stLuna holders")
	ctx := util.PrepCtx(app)

	app.Logger().Info("... stLUNA")
	bondedStLunaHolders := make(map[string]sdk.Int)
	err := util.GetCW20AccountsAndBalances(ctx, app.WasmKeeper, StLuna, bondedStLunaHolders)
	if err != nil {
		return err
	}
	snapshot.Add(bondedStLunaHolders, util.DenomSTLUNA)
	snapshot.ApplyBlackList(bl)

	app.Logger().Info("... bLUNA")
	bondedBLunaHolders := make(map[string]sdk.Int)
	err = util.GetCW20AccountsAndBalances(ctx, app.WasmKeeper, BLuna, bondedBLunaHolders)
	if err != nil {
		return err
	}
	snapshot.Add(bondedBLunaHolders, util.DenomBLUNA)

	return nil
}

func ResolveLidoLuna(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap, bl util.Blacklist) error {
	app.Logger().Info("Resolving bLuna and stLuna to LUNA")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	lidoState, err := getExchangeRates(ctx, q)
	if err != nil {
		return err
	}
	snapshot.ApplyBlackList(bl)

	for _, sbs := range snapshot {
		for i, sb := range sbs {
			if sb.Denom == util.DenomBLUNA {
				sbs[i] = util.SnapshotBalance{
					Denom:   util.DenomLUNA,
					Balance: lidoState.StLunaExchangeRate.MulInt(sb.Balance).TruncateInt(),
				}
			}
			if sb.Denom == util.DenomSTLUNA {
				sbs[i] = util.SnapshotBalance{
					Denom:   util.DenomLUNA,
					Balance: lidoState.BLunaExchangeRate.MulInt(sb.Balance).TruncateInt(),
				}
			}
		}
	}
	unbondingBluna, unbondingStLuna, err := getUnbondingTokens(ctx, app.WasmKeeper)
	if err != nil {
		return nil
	}
	unbondingLuna := util.MergeMaps(applyExchangeRates(unbondingBluna, lidoState.BLunaExchangeRate), applyExchangeRates(unbondingStLuna, lidoState.StLunaExchangeRate))
	bl.RegisterAddress(util.DenomLUNA, LidoHub)
	snapshot.Add(unbondingLuna, util.DenomLUNA)
	return nil
}

func ExportLidoRewards(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap, bl util.Blacklist) error {
	app.Logger().Info("Distributing Lido staking rewards")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	snapshot.ApplyBlackList(bl)

	bondedBLunaHolders := make(map[string]sdk.Int)
	bondedStLunaHolders := make(map[string]sdk.Int)

	for acc, sbs := range snapshot {
		for _, sb := range sbs {
			if sb.Denom == util.DenomBLUNA {
				if bondedBLunaHolders[acc].IsNil() {
					bondedBLunaHolders[acc] = sb.Balance
				} else {
					bondedBLunaHolders[acc] = bondedBLunaHolders[acc].Add(sb.Balance)
				}
			}
			if sb.Denom == util.DenomSTLUNA {
				if bondedStLunaHolders[acc].IsNil() {
					bondedStLunaHolders[acc] = sb.Balance
				} else {
					bondedStLunaHolders[acc] = bondedStLunaHolders[acc].Add(sb.Balance)
				}
			}
		}
	}

	lidoState, err := getExchangeRates(ctx, q)
	if err != nil {
		return err
	}

	stLunaTotalSupply, err := util.GetCW20TotalSupply(ctx, q, StLuna)
	if err != nil {
		return err
	}
	bLunaTotalSupply, err := util.GetCW20TotalSupply(ctx, q, BLuna)
	if err != nil {
		return err
	}

	err = util.AlmostEqual("stLUNA", snapshot.SumOfDenom(util.DenomSTLUNA), stLunaTotalSupply, sdk.NewInt(100000))
	if err != nil {
		app.Logger().Debug(err.Error())
	}
	err = util.AlmostEqual("bLUNA", snapshot.SumOfDenom(util.DenomBLUNA), bLunaTotalSupply, sdk.NewInt(100000))
	if err != nil {
		app.Logger().Debug(err.Error())
	}

	lunaRewards, err := getLunaRewards(ctx, app.BankKeeper)
	if err != nil {
		return err
	}

	stLunaRewards := lunaRewards.Mul(lidoState.TotalBondStLuna.Quo(lidoState.TotalBondStLuna.Add(lidoState.TotalBondBLuna)))
	bLunaRewards := lunaRewards.Mul(lidoState.TotalBondBLuna.Quo(lidoState.TotalBondStLuna.Add(lidoState.TotalBondBLuna)))

	ustRewards, err := getUstRewards(ctx, app.BankKeeper)
	if err != nil {
		return err
	}
	stLunaUstRewards := ustRewards.Mul(lidoState.TotalBondStLuna.Quo(lidoState.TotalBondStLuna.Add(lidoState.TotalBondBLuna)))
	bLunaUstRewards := ustRewards.Mul(lidoState.TotalBondBLuna.Quo(lidoState.TotalBondStLuna.Add(lidoState.TotalBondBLuna)))

	lunaBalance := make(map[string]sdk.Int)
	for k, _ := range bondedStLunaHolders {
		if lunaBalance[k].IsNil() {
			lunaBalance[k] = stLunaRewards.Mul(bondedStLunaHolders[k].Quo(stLunaTotalSupply))
		} else {
			lunaBalance[k] = lunaBalance[k].Add(stLunaRewards.Mul(bondedStLunaHolders[k].Quo(stLunaTotalSupply)))
		}
	}
	for k, _ := range bondedBLunaHolders {
		if lunaBalance[k].IsNil() {
			lunaBalance[k] = bLunaRewards.Mul(bondedBLunaHolders[k].Quo(bLunaTotalSupply))
		} else {
			lunaBalance[k] = lunaBalance[k].Add(bLunaRewards.Mul(bondedBLunaHolders[k].Quo(bLunaTotalSupply)))
		}
	}

	ustBalance := make(map[string]util.SnapshotBalance)
	for k, _ := range bondedStLunaHolders {
		ustToAdd := stLunaUstRewards.Mul(bondedStLunaHolders[k].Quo(stLunaTotalSupply))
		if ustToAdd.IsZero() {
			continue
		}
		if ustBalance[k].Balance.IsNil() {
			ustBalance[k] = util.SnapshotBalance{
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
			ustBalance[k] = util.SnapshotBalance{
				Denom:   util.DenomUST,
				Balance: sdk.NewInt(0),
			}
		}
		userUstBalance := ustBalance[k]
		(&userUstBalance).AddInto(ustToAdd)
	}
	return nil
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
		lunaBalances[k] = exchangeRate.MulInt(v).TruncateInt()
	}
	return lunaBalances
}

func getUnbondingTokens(ctx context.Context, k wasmkeeper.Keeper) (map[string]sdk.Int, map[string]sdk.Int, error) {
	prefix := util.GeneratePrefix("v2_wait")
	lidoHubAddr, err := sdk.AccAddressFromBech32(LidoHub)
	if err != nil {
		panic(err)
	}
	bLunaHolding := make(map[string]sdk.Int)
	stLunaHolding := make(map[string]sdk.Int)
	k.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), lidoHubAddr, prefix, func(key, value []byte) bool {
		// key is in the format [len("\"address\"")]["address"][1 byte]
		var unbondingRes struct {
			BLunaAmount  sdk.Int `json:"bluna_amount"`
			StLunaAmount sdk.Int `json:"stluna_amount"`
		}
		wallet := string(key[3 : len(key)-4])
		err = json.Unmarshal(value, &unbondingRes)
		if err != nil {
			panic(err)
		}
		// Users can have multiple unbounding requests
		if !unbondingRes.BLunaAmount.IsZero() {
			if bLunaHolding[wallet].IsNil() {
				bLunaHolding[wallet] = sdk.NewInt(0)
			}
			bLunaHolding[wallet] = bLunaHolding[wallet].Add(unbondingRes.BLunaAmount)
		}
		if !unbondingRes.StLunaAmount.IsZero() {
			if stLunaHolding[wallet].IsNil() {
				stLunaHolding[wallet] = sdk.NewInt(0)
			}
			stLunaHolding[wallet] = stLunaHolding[wallet].Add(unbondingRes.StLunaAmount)
		}
		return false
	})
	return bLunaHolding, stLunaHolding, nil
}
