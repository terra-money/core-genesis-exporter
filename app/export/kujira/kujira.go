package kujira

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
)

const (
	KujiraStaking   = "terra1cf9q9lq7tdfju95sdw78y9e34a6qrq3rrc6dre"
	KujiraAUstVault = "terra13nk2cjepdzzwfqy740pxzpe3x75pd6g0grxm2z"
	KujiraUstLP     = "terra1cmqv3sjew8kcm3j907x2026e4n0ejl2jackxlx"
	KujiraUstPair   = "terra1zkyrfyq7x9v5vqnnrznn3kvj35az4f6jxftrl2"
)

func ExportKujiraVault(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap, bl *util.Blacklist) error {
	app.Logger().Info("Exporting Kujira vaults")
	ctx := util.PrepCtx(app)
	prefix := util.GeneratePrefix("bid")
	vaultAddr, err := sdk.AccAddressFromBech32(KujiraAUstVault)
	if err != nil {
		return err
	}

	balances := make(map[string]sdk.Int)
	app.WasmKeeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), vaultAddr, prefix, func(key, value []byte) bool {
		var bid struct {
			Bidder       string  `json:"bidder"`
			Amount       sdk.Int `json:"amount"`
			ExchangeRate sdk.Dec `json:"prev_exchange_rate"`
		}
		err := json.Unmarshal(value, &bid)
		if err != nil {
			panic(err)
		}
		if bid.Amount.IsZero() {
			return false
		}

		bidderAddr, err := util.AccAddressFromBase64(bid.Bidder)
		if err != nil {
			panic(err)
		}

		if balances[bidderAddr.String()].IsNil() {
			balances[bidderAddr.String()] = bid.Amount
		} else {
			balances[bidderAddr.String()] = balances[bidderAddr.String()].Add(bid.Amount)
		}
		return false
	})

	q := util.PrepWasmQueryServer(app)
	vaultBalance, err := util.GetCW20Balance(ctx, q, util.AUST, KujiraAUstVault)
	if err != nil {
		return err
	}
	fmt.Printf("total in vault: %s\n", vaultBalance)

	// Small rounding error (.00006%) here due to the way Kujira saves amount of aUST deposited
	// When converting aUST to UST, the anchor exchange rate is used instead of
	// listening to the hook of the new UST balance
	err = util.AlmostEqual("kujira aUST", vaultBalance, util.Sum(balances), sdk.NewInt(50000000))
	if err != nil {
		return err
	}
	bl.RegisterAddress(util.DenomAUST, KujiraAUstVault)
	snapshot.Add(balances, util.DenomAUST)
	return nil
}

// IGNORE: should be accounted for by terraswap side
func ExportKujiraStaking(app *terra.TerraApp, bl *util.Blacklist) (map[string]map[string]sdk.Int, error) {
	ctx := util.PrepCtx(app)
	prefix := util.GeneratePrefix("reward")
	stakeAddr, err := sdk.AccAddressFromBech32(KujiraStaking)
	if err != nil {
		return nil, err
	}

	lpBalances := make(map[string]map[string]sdk.Int)
	balances := make(map[string]sdk.Int)
	app.WasmKeeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), stakeAddr, prefix, func(key, value []byte) bool {
		var reward struct {
			Amount sdk.Int `json:"bond_amount"`
		}
		json.Unmarshal(value, &reward)
		holderAddr := sdk.AccAddress(key)
		balances[holderAddr.String()] = reward.Amount
		return false
	})

	q := util.PrepWasmQueryServer(app)
	vaultBalance, err := util.GetCW20Balance(ctx, q, KujiraUstLP, KujiraStaking)
	if err != nil {
		return nil, err
	}

	sumVault := sdk.NewInt(0)
	for _, b := range balances {
		sumVault = sumVault.Add(b)
	}

	fmt.Printf("LP in staking: %s, sum of depositors: %s, difference: %s\n", vaultBalance, sumVault, vaultBalance.Sub(sumVault))
	bl.RegisterAddress(util.DenomUST, KujiraUstPair)
	bl.RegisterAddress(KujiraUstLP, KujiraStaking)
	lpBalances[KujiraUstLP] = balances
	return lpBalances, nil
}
