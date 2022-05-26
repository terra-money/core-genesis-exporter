package angel

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	wasmtypes "github.com/terra-money/core/x/wasm/types"

	"github.com/terra-money/core/app/export/util"
)

const (
	Endowments = "terra1nwk2y5nfa5sxx6gtxr84lre3zpnn7cad2f266h"
	APANC      = "terra172ue5d0zm7jlsj2d9af4vdff6wua7mnv6dq5vp"
	DANO       = "terra1rcznds2le2eflj3y4e8ep3e4upvq04sc65wdly" // Wallet that holds UST/aUST for charities.
)

// ExportEndowments Export aUST endowments
func ExportEndowments(app *terra.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	snapshot := make(util.SnapshotBalanceAggregateMap)
	logger := app.Logger()
	logger.Info("Exporting Angel Protocol endowments")

	totalaUST := sdk.NewInt(0)

	var endowments struct {
		Endowments []struct {
			Address string `json:"address"`
		} `json:"endowments"`
	}

	if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: Endowments,
		QueryMsg:        []byte("{\"endowment_list\":{}}"),
	}, &endowments); err != nil {
		return nil, err
	}

	for _, endowment := range endowments.Endowments {
		var apANCBalance struct {
			LockedCW20 []struct {
				Address string  `json:"address"`
				Amount  sdk.Int `json:"amount"`
			} `json:"locked_cw20"`
			LiquidCW20 []struct {
				Address string  `json:"address"`
				Amount  sdk.Int `json:"amount"`
			} `json:"liquid_cw20"`
		}

		if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: APANC,
			QueryMsg:        []byte(fmt.Sprintf("{\"balance\":{\"address\":\"%s\"}}", endowment.Address)),
		}, &apANCBalance); err != nil {
			return nil, err
		}

		aUSTBalance := sdk.NewInt(0)

		if len(apANCBalance.LiquidCW20) != 0 {
			aUSTBalance = aUSTBalance.Add(apANCBalance.LiquidCW20[0].Amount)
		}

		if len(apANCBalance.LockedCW20) != 0 {
			aUSTBalance = aUSTBalance.Add(apANCBalance.LockedCW20[0].Amount)
		}

		if aUSTBalance.IsZero() {
			continue
		}

		totalaUST = totalaUST.Add(aUSTBalance)

		// Fetch endowment owner from InitMsg.
		var initMsg struct {
			Owner string `json:"owner_sc"`
		}

		if err := util.ContractInitMsg(ctx, q, &wasmtypes.QueryContractInfoRequest{
			ContractAddress: endowment.Address,
		}, &initMsg); err != nil {
			return nil, err
		}

		snapshot.AppendOrAddBalance(initMsg.Owner, util.SnapshotBalance{
			Denom:   util.DenomAUST,
			Balance: aUSTBalance,
		})
	}

	logger.Info(fmt.Sprintf("total aUST indexed: %d", totalaUST.Int64()))

	// These balances are counted using apANC tokens above.
	bl.RegisterAddress(util.DenomUST, DANO)
	bl.RegisterAddress(util.DenomAUST, DANO)

	return snapshot, nil
}
