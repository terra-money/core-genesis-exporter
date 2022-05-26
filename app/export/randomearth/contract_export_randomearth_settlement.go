package randomearth

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"

	"github.com/terra-money/core/app/export/util"
)

const (
	Settlement = "terra1eek0ymmhyzja60830xhzm7k7jkrk99a60q2z2t"
)

// ExportRandomEarthSettlements Index Luna held in RandomEarth settlement contract.
func ExportRandomEarthSettlements(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap, bl *util.Blacklist) error {
	ctx := util.PrepCtx(app)
	holdings := make(util.BalanceMap)

	logger := app.Logger()
	logger.Info("fetching RandomEarth settlement balances...")

	// Pull users from balances map.
	// pub const BALANCES: Map<(&[u8], &[u8]), Uint128> = Map::new("balances");
	prefix := util.GeneratePrefix("balances")
	delegatorAddr, err := sdk.AccAddressFromBech32(Settlement)
	if err != nil {
		return err
	}

	app.WasmKeeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), delegatorAddr, prefix, func(key, value []byte) bool {
		// We only care about uluna balances. This map also includes NFTs and other holdings.
		if strings.Contains(string(key), "uluna") {
			// Filter out characters from start and end of the key.
			correctedAddress := string(key)[2:46]
			// Remove quotes from the value and convert to an Int.
			balance, ok := sdk.NewIntFromString(strings.Trim(string(value), "\""))
			if ok && !balance.IsZero() {
				previousAmount := holdings[correctedAddress]
				if previousAmount.IsNil() {
					previousAmount = sdk.NewInt(0)
				}

				holdings[correctedAddress] = previousAmount.Add(balance)
			}
		}

		return false
	})

	snapshot.Add(holdings, util.DenomLUNA)
	bl.RegisterAddress(util.DenomLUNA, Settlement)

	return nil
}
