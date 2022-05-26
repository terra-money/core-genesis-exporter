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
func ExportIDO(app *terra.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	logger := app.Logger()
	logger.Info("Exporting StarTerra IDO balances")

	snapshot := make(util.SnapshotBalanceAggregateMap)

	var idoFunders struct {
		Users []struct {
			Funder         string  `json:"funder"`
			AvailableFunds sdk.Int `json:"available_funds"`
		} `json:"users"`
	}

	var offset = ""
	for {
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

			snapshot.AppendOrAddBalance(userInfo.Funder, util.SnapshotBalance{
				Denom:   util.DenomUST,
				Balance: userInfo.AvailableFunds,
			})
		}

		offset = idoFunders.Users[len(idoFunders.Users)-1].Funder
	}

	bl.RegisterAddress(util.DenomUST, IDO)
	return snapshot, nil
}
