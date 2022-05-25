package angel

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	wasmtypes "github.com/terra-money/core/x/wasm/types"

	"github.com/terra-money/core/app/export/util"
)

const (
	AngelEndowments = "terra1nwk2y5nfa5sxx6gtxr84lre3zpnn7cad2f266h"
	AngelAPANC      = "terra172ue5d0zm7jlsj2d9af4vdff6wua7mnv6dq5vp"
)

// ExportAngelEndowments Export aUST endowments
func ExportAngelEndowments(app *terra.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	balances := make(util.SnapshotBalanceMap)

	var endowments struct {
		Endowments []struct {
			Address string `json:"address"`
		} `json:"endowments"`
	}

	if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: AngelEndowments,
		QueryMsg:        []byte("{\"endowment_list\":{}}"),
	}, &endowments); err != nil {
		panic(err)
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
			ContractAddress: AngelAPANC,
			QueryMsg:        []byte(fmt.Sprintf("{\"balance\":{\"address\":\"%s\"}}", endowment.Address)),
		}, &apANCBalance); err != nil {
			panic(err)
		}

		// TODO: Need to confirm aUST <-> apANC exchange rate.
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

		previousAmount := balances[endowment.Address].Balance
		if previousAmount.IsNil() {
			previousAmount = sdk.NewInt(0)
		}

		balances[endowment.Address] = util.SnapshotBalance{
			Denom:   util.DenomAUST,
			Balance: previousAmount.Add(aUSTBalance),
		}
	}

	return balances, nil
}
