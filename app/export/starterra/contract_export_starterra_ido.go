package starterra

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	wasmtypes "github.com/terra-money/core/x/wasm/types"

	"github.com/terra-money/core/app/export/util"
)

const (
	IDO = "terra1yzewp648fwq7ymlfdg5h90dfzk5y2hf6kk9pdm"
)

// ExportIDO Export unspent funds from StarTerra IDO platform.
// Even though users deposit UST, the protocol changes some of it to aUST
// When we calculate ownership, we will split all the funds in the IDO back to the users
// Users should obtain a mix of UST and aUST to simply calculation
func ExportIDO(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	logger := app.Logger()
	logger.Info("Exporting StarTerra IDO balances")

	shareHoldings := make(util.BalanceMap)
	var offset = ""
	for {
		var idoFunders struct {
			Users []struct {
				Funder         string  `json:"funder"`
				AvailableFunds sdk.Int `json:"available_funds"`
			} `json:"users"`
		}
		query := "{\"funders\": {\"limit\": 1024}}"
		if offset != "" {
			query = fmt.Sprintf("{\"funders\": {\"start_after\": \"%s\", \"limit\": 1024}}", offset)
		}

		if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: IDO,
			QueryMsg:        []byte(query),
		}, &idoFunders); err != nil {
			return nil, err
		}

		if len(idoFunders.Users) == 0 {
			break
		}

		for _, userInfo := range idoFunders.Users {
			if userInfo.AvailableFunds.IsZero() {
				continue
			}

			if shareHoldings[userInfo.Funder].IsNil() {
				shareHoldings[userInfo.Funder] = userInfo.AvailableFunds
			} else {
				shareHoldings[userInfo.Funder] = shareHoldings[userInfo.Funder].Add(userInfo.AvailableFunds)
			}
		}

		offset = idoFunders.Users[len(idoFunders.Users)-1].Funder
	}
	ustBalance, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomUST, IDO)
	if err != nil {
		return nil, err
	}
	aUstBalance, err := util.GetCW20Balance(ctx, q, util.AUST, IDO)
	if err != nil {
		return nil, err
	}
	totalShares := util.Sum(shareHoldings)
	ustRatio := sdk.NewDecFromInt(ustBalance).QuoInt(totalShares)
	aUstRatio := sdk.NewDecFromInt(aUstBalance).QuoInt(totalShares)

	snapshot := make(util.SnapshotBalanceAggregateMap)
	for addr, balance := range shareHoldings {
		snapshot.AppendOrAddBalance(addr, util.SnapshotBalance{
			Denom:   util.DenomUST,
			Balance: ustRatio.MulInt(balance).TruncateInt(),
		})
		snapshot.AppendOrAddBalance(addr, util.SnapshotBalance{
			Denom:   util.DenomAUST,
			Balance: aUstRatio.MulInt(balance).TruncateInt(),
		})
	}

	bl.RegisterAddress(util.DenomUST, IDO)
	bl.RegisterAddress(util.DenomAUST, IDO)
	return snapshot, nil
}

func Audit(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	ustBalance, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomUST, IDO)
	if err != nil {
		return err
	}

	err = util.AlmostEqual("starterra ido: ust", snapshot.SumOfDenom(util.DenomUST), ustBalance, sdk.NewInt(10000))
	if err != nil {
		return err
	}

	aUstBalance, err := util.GetCW20Balance(ctx, q, util.AUST, IDO)
	if err != nil {
		return err
	}

	err = util.AlmostEqual("starterra ido: aust", snapshot.SumOfDenom(util.DenomAUST), aUstBalance, sdk.NewInt(10000))
	if err != nil {
		return err
	}

	return nil
}
