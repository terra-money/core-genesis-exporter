package kujira

import (
	"encoding/json"

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

func ExportKujiraVault(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Kujira vaults")
	ctx := util.PrepCtx(app)
	prefix := util.GeneratePrefix("bid")
	vaultAddr, err := sdk.AccAddressFromBech32(KujiraAUstVault)
	if err != nil {
		return nil, err
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

	snapshot := make(util.SnapshotBalanceAggregateMap)
	bl.RegisterAddress(util.DenomAUST, KujiraAUstVault)
	snapshot.Add(balances, util.DenomAUST)
	return snapshot, nil
}

func Audit(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	vaultBalance, err := util.GetCW20Balance(ctx, q, util.AUST, KujiraAUstVault)
	if err != nil {
		return err
	}
	// Small rounding error (.00006%) here due to the way Kujira saves amount of aUST deposited
	// When converting aUST to UST, the anchor exchange rate is used instead of
	// listening to the hook of the new UST balance
	util.AlmostEqual("kujira aUST", vaultBalance, snapshot.SumOfDenom(util.DenomAUST), sdk.NewInt(100000000))
	return nil
}
