package starterra

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	wasmtypes "github.com/terra-money/core/x/wasm/types"

	"github.com/terra-money/core/app/export/util"
)

const (
	StarTerraIDO = "terra1yzewp648fwq7ymlfdg5h90dfzk5y2hf6kk9pdm"
)

// ExportStarTerraIDO Export unspent funds from StarTerra IDO platform.
func ExportStarTerraIDO(app *terra.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	balances := make(util.SnapshotBalanceMap)

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
			ContractAddress: StarTerraIDO,
			QueryMsg:        []byte(query),
		}, &idoFunders); err != nil {
			panic(err)
		}

		if len(idoFunders.Users) == 0 {
			break
		}

		for _, userInfo := range idoFunders.Users {
			if userInfo.AvailableFunds.IsZero() {
				continue
			}

			previousAmount := balances[userInfo.Funder].Balance
			if previousAmount.IsNil() {
				previousAmount = sdk.NewInt(0)
			}

			balances[userInfo.Funder] = util.SnapshotBalance{
				Denom:   util.DenomUST,
				Balance: previousAmount.Add(userInfo.AvailableFunds),
			}
		}

		offset = idoFunders.Users[len(idoFunders.Users)-1].Funder
	}

	bl.RegisterAddress(util.DenomUST, StarTerraIDO)
	return balances, nil
}
