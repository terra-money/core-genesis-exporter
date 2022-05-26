package aperture

import (
	"context"
	"fmt"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	log "github.com/tendermint/tendermint/libs/log"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	apertureManager             = "terra1ajkmy2c0g84seh66apv9x6xt6kd3ag80jmcvtz"
	apertureDeltaNeutralManager = "terra1jvehz6d9gk3gl4tldrzd8qzj8zfkurfvtcg99x"
	positionContractAddress     = "terra1aaxjgzems5gm07mg6fa50mjxjevsx4gsk6np5j"
)

type BatchResponse struct {
	Items []BatchItem `json:"items"`
}

type BatchItem struct {
	Contract string `json:"contract"`
	Holder   string `json:"holder"`
	Info     struct {
		PositionOpenInfo struct {
			Height int `json:"height"`
		} `json:"position_open_info"`
		PositionCloseInfo struct {
			Height int `json:"height"`
		} `json:"position_close_info"`
		DetailedInfo struct {
			State struct {
				AUstAmount sdk.Int `json:"collateral_anchor_ust_amount"`
			} `json:"state"`
		} `json:"detailed_info"`
		UstAmount sdk.Int `json:"uusd_value"`
	} `json:"info"`
}

func ExportApertureVaults(app *terra.TerraApp, snapshotType util.Snapshot, bl *util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Aperture (this takes a while)")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	lastPosition, err := getApertureLastPositionId(ctx, q)
	if err != nil {
		return nil, err
	}
	workerCount := 2
	jobs := make(chan (sdk.Int), lastPosition.Int64())
	for j := int64(0); j < lastPosition.Int64(); j++ {
		jobs <- sdk.NewInt(j)
	}
	close(jobs)
	items := make(chan (BatchItem), lastPosition.Int64())
	wg := sync.WaitGroup{}
	for i := 0; i < workerCount; i++ {
		wg.Add(i)
		go worker(&wg, app.Logger(), ctx, q, jobs, items)
	}
	wg.Wait()
	close(items)
	var balances = make(map[string]map[string]sdk.Int)
	for item := range items {
		// Avoid double counting by only taking aUST amount for pre-attack snapshot
		// UST amount in aperture is a "virtual" amount as the UST is converted to aUST and used
		// as collateral in mirror. The UST amount field is a calculated field for the final UST amount
		// owned by the wallet
		if snapshotType == util.Snapshot(util.PreAttack) {
			balances[util.AUST][item.Holder] = item.Info.DetailedInfo.State.AUstAmount
			bl.RegisterAddress(util.DenomAUST, item.Contract)
			fmt.Println(item)
		} else {
			balances["uusd"][item.Holder] = item.Info.UstAmount
		}
	}
	snapshot := make(util.SnapshotBalanceAggregateMap)
	snapshot.Add(balances[util.DenomAUST], util.DenomAUST)
	snapshot.Add(balances[util.DenomUST], util.DenomUST)
	return snapshot, nil
}

func getApertureLastPositionId(ctx context.Context, q wasmtypes.QueryServer) (sdk.Int, error) {
	var nextPositionResponse struct {
		NextPosition sdk.Int `json:"next_position_id"`
	}
	err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: apertureManager,
		QueryMsg:        []byte("{\"get_next_position_id\": {}}"),
	}, &nextPositionResponse)
	if err != nil {
		return nextPositionResponse.NextPosition, err
	}
	return nextPositionResponse.NextPosition, nil
}

func worker(wg *sync.WaitGroup, log log.Logger, ctx context.Context, q wasmtypes.QueryServer, jobs <-chan (sdk.Int), results chan<- (BatchItem)) {
	defer wg.Done()
	for j := range jobs {
		err := getApertureOpenPositions(log, ctx, q, j, results)
		if err != nil {
			panic(err)
		}
	}
}

func getApertureOpenPositions(log log.Logger, ctx context.Context, q wasmtypes.QueryServer, positionId sdk.Int, results chan<- (BatchItem)) error {
	if positionId.ModRaw(500).IsZero() {
		log.Info(fmt.Sprintf("... Position %s", positionId))
	}
	var batchResponse BatchResponse
	positionQuery := fmt.Sprintf("{\"position_id\":\"%s\", \"chain_id\": 3 }", positionId)
	query := fmt.Sprintf("{\"batch_get_position_info\": {\"positions\": [%s]}}", positionQuery)
	// fmt.Println(query)
	err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: apertureDeltaNeutralManager,
		QueryMsg:        []byte(query),
	}, &batchResponse)
	if err != nil {
		return err
	}
	for _, item := range batchResponse.Items {
		if item.Info.PositionCloseInfo.Height == 0 {
			results <- item
		}
	}
	if err != nil {
		panic(err)
	}
	return nil
}
