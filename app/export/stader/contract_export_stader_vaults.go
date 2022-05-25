package stader

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	wasmtypes "github.com/terra-money/core/x/wasm/types"

	"github.com/terra-money/core/app/export/util"
)

const (
	Vaults = "terra1v05vafsr8w8ar0mw040cluz0rq6pg2rrpues5r"
)

// ExportStaderVaults Export LunaX balances in Stader vaults.
func ExportStaderVaults(app *terra.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	balances := make(util.SnapshotBalanceMap)

	logger := app.Logger()
	logger.Info("fetching Stader vault balances...")

	exchangeRate, err := GetLunaXExchangeRate(ctx, q)
	if err != nil {
		return nil, err
	}

	var staderVaults struct {
		UserDetails []struct {
			Address      string  `json:"address"`
			DepositValue sdk.Int `json:"deposit_value"`
		} `json:"user_details"`
	}

	var offset = ""
	for {
		query := "{\"users\": {\"limit\": 30}}"
		if offset != "" {
			query = fmt.Sprintf("{\"users\": {\"start_after\": \"%s\", \"limit\": 30}}", offset)
		}

		if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: Vaults,
			QueryMsg:        []byte(query),
		}, &staderVaults); err != nil {
			panic(err)
		}

		if len(staderVaults.UserDetails) == 0 {
			break
		}

		for _, userDetails := range staderVaults.UserDetails {
			previousAmount := balances[userDetails.Address].Balance
			if previousAmount.IsNil() {
				previousAmount = sdk.NewInt(0)
			}

			balances[userDetails.Address] = util.SnapshotBalance{
				Denom:   util.DenomLUNA,
				Balance: previousAmount.Add(exchangeRate.MulInt(userDetails.DepositValue).TruncateInt()),
			}
		}

		offset = staderVaults.UserDetails[len(staderVaults.UserDetails)-1].Address
	}

	return balances, nil
}
