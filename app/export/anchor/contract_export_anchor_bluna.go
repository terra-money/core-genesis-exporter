package anchor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
)

// ExportbLUNA get bLUNA provided to anchor as collateral.
// ER conversion is taken later in lido exporter
func ExportbLUNA(app *app.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	bl.RegisterAddress(util.DenomLUNA, AddressBLUNAHub)

	ctx := util.PrepCtx(app)
	logger := app.Logger()

	var finalBalance = make(util.SnapshotBalanceAggregateMap)
	logger.Info("fetching bLUNA provided in anchor custody...")

	// iterate over all provided bLUNA in anchor
	collateralPrefix := util.GeneratePrefix("collateral")
	keeper := app.WasmKeeper
	overseer, _ := sdk.AccAddressFromBech32(AddressOverseer)

	logger.Info("fetching bLUNA provided in overseer...")
	var collaterals = make([][2]string, 0)
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), overseer, collateralPrefix, func(key, value []byte) bool {
		userAddr := sdk.AccAddress(key).String()
		if err := json.Unmarshal(value, &collaterals); err != nil {
			panic(fmt.Errorf("err while fetching provided bluna: %v", err))
		}

		// filter only bLUNA
		for _, collateral := range collaterals {
			bz, _ := base64.StdEncoding.DecodeString(collateral[0])
			assetAddr := sdk.AccAddress(bz).String()

			if assetAddr != util.AddressBLUNA {
				continue
			}

			balance, _ := sdk.NewIntFromString(collateral[1])

			finalBalance.AppendOrAddBalance(userAddr, util.SnapshotBalance{
				Denom:   util.DenomBLUNA,
				Balance: balance,
			})
		}

		return false
	})

	// exchange rate is handled in lido code
	return finalBalance, nil
}
