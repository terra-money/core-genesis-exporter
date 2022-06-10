package oneplanet

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	terra "github.com/terra-money/core/app"

	"github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

type Contract struct {
	Address string
	Denom   string
}

var (
	opUST = Contract{
		Address: "terra1r27hqy58tmgnv9uykc708wzdlel9n3g0qdm04c",
		Denom:   util.DenomUST,
	}
	opLUNA = Contract{
		Address: "terra15d2d3pw6ag3tltycmvn2uxfnlwcr86utyl83r6",
		Denom:   util.DenomLUNA,
	}
)

// ExportHoldings Index holdings in OnePlanet storage contracts (opluna and opust).
func ExportHoldings(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting OnePlanet")
	var _ wasmtypes.QueryServer
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	snapshot := make(util.SnapshotBalanceAggregateMap)

	for _, contract := range []Contract{opUST, opLUNA} {
		balances := make(util.BalanceMap)

		err := util.GetCW20AccountsAndBalances(ctx, app.WasmKeeper, contract.Address, balances)
		if err != nil {
			return nil, err
		}

		for address, amount := range balances {
			if amount.IsZero() {
				continue
			}

			err, owner := getOwnerIfExists(ctx, q, address)
			if err != nil {
				// Not a contract, attribute funds to user wallet.
				snapshot.AppendOrAddBalance(address, util.SnapshotBalance{
					Denom:   contract.Denom,
					Balance: amount,
				})

				continue
			}

			if owner == "" {
				var salesInitMsg struct {
					Config struct {
						Recipient string `json:"recipient_wallet"`
					} `json:"config"`
				}
				if err := util.ContractInitMsg(ctx, q, &wasmtypes.QueryContractInfoRequest{
					ContractAddress: address,
				}, &salesInitMsg); err != nil {
					return nil, err
				}

				err, owner = getOwnerIfExists(ctx, q, salesInitMsg.Config.Recipient)
				if err != nil {
					owner = salesInitMsg.Config.Recipient
				}
			}

			snapshot.AppendOrAddBalance(owner, util.SnapshotBalance{
				Denom:   contract.Denom,
				Balance: amount,
			})
		}
	}

	return snapshot, nil
}

func getOwnerIfExists(ctx context.Context, q wasmtypes.QueryServer, contractAddress string) (error, string) {
	var initMsg struct {
		Owner string `json:"owner"`
	}

	if err := util.ContractInitMsg(ctx, q, &wasmtypes.QueryContractInfoRequest{
		ContractAddress: contractAddress,
	}, &initMsg); err != nil {
		return err, ""
	}

	return nil, initMsg.Owner
}

func Audit(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	app.Logger().Info("Audit -- OnePlanet")
	ctx := util.PrepCtx(app)

	ustBalance, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomUST, opUST.Address)
	if err != nil {
		return err
	}

	if err := util.AlmostEqual(util.DenomUST, ustBalance, snapshot.SumOfDenom(util.DenomUST), sdk.NewInt(1000000)); err != nil {
		return err
	}

	lunaBalance, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomLUNA, opLUNA.Address)
	if err != nil {
		return err
	}

	if err := util.AlmostEqual(util.DenomLUNA, lunaBalance, snapshot.SumOfDenom(util.DenomLUNA), sdk.NewInt(1000000)); err != nil {
		return err
	}

	return nil
}
