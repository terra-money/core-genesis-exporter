package app

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	apolloFactory = "terra1g7jjjkt5uvkjeyhp8ecdz4e4hvtn83sud3tmh2"
)

type Strategy struct {
	Id              int    `json:"id"`
	Address         string `json:"address"`
	TotalBondAmount string `json:"total_bond_amount"`
}

func ExportApolloVaultLPs(app *TerraApp, q wasmtypes.QueryServer) (map[string]map[string]sdk.Int, error) {
	height := app.LastBlockHeight()
	log.Println(height)
	ctx := prepCtx(app)
	_, err := getListOfStrategies(ctx, q, 1, 0)
	if err != nil {
		log.Println(err)
	}
	return nil, nil
}

func getListOfStrategies(ctx context.Context, q wasmtypes.QueryServer, limit, offset int) ([]Strategy, error) {
	query := fmt.Sprintf("{\"get_strategies\": {\"limit\":%d, \"start_from\":%d}}", limit, offset)
	var strategies struct {
		Strategies []Strategy
	}
	if err := contractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: apolloFactory,
		QueryMsg:        []byte(query),
	}, &strategies); err != nil {
		return []Strategy{}, err
	}
	fmt.Printf("%v", strategies)
	return strategies.Strategies, nil
}
