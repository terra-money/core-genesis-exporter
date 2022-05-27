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

// ExportVaults Export LunaX balances in Stader vaults.
func ExportVaults(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	snapshot := make(util.SnapshotBalanceAggregateMap)
	logger := app.Logger()
	logger.Info("Exporting Stader vault balances")

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
			return nil, err
		}

		if len(staderVaults.UserDetails) == 0 {
			break
		}

		for _, userDetails := range staderVaults.UserDetails {
			var userClaims struct {
				ClaimAmount sdk.Int `json:"claim_amount"`
				HasClaimed  bool    `json:"has_claimed"`
			}
			if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
				ContractAddress: Vaults,
				QueryMsg:        []byte(fmt.Sprintf("{\"user_claim\": {\"address\": \"%s\"}}", userDetails.Address)),
			}, &userClaims); err != nil {
				return nil, err
			}

			if !userClaims.HasClaimed {
				snapshot.AppendOrAddBalance(userDetails.Address, util.SnapshotBalance{
					Denom:   util.DenomLUNA,
					Balance: exchangeRate.MulInt(userClaims.ClaimAmount).TruncateInt(),
				})
			}
		}

		offset = staderVaults.UserDetails[len(staderVaults.UserDetails)-1].Address
	}

	return snapshot, nil
}
