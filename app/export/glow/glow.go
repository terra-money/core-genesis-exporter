package glow

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/anchor"
	"github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

const (
	GlowLotto = "terra1tu9yjssxslh3fd6fe908ntkquf3nd3xt8kp2u2"
)

type Deposit struct {
	Depositor      string  `json:"depositor"`
	LotteryDeposit sdk.Int `json:"lottery_deposit"`
	Savings        sdk.Int `json:"savings_aust"`
}

func ExportContract(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Glow Lotto")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	var depositorQuery func(string) error
	limit := 50
	var allDeposits []Deposit
	depositorQuery = func(startAfter string) error {
		var deposits struct {
			Depositors []Deposit `json:"depositors"`
		}
		var query string
		if startAfter == "" {
			query = fmt.Sprintf("{\"depositors\": {\"limit\": %d}}", limit)
		} else {
			query = fmt.Sprintf("{\"depositors\": {\"limit\": %d, \"start_after\": \"%s\"}}", limit, startAfter)
		}
		err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: GlowLotto,
			QueryMsg:        []byte(query),
		}, &deposits)

		if err != nil {
			return err
		}
		allDeposits = append(allDeposits, deposits.Depositors...)
		if len(deposits.Depositors) < limit {
			return nil
		}
		lastAddress := deposits.Depositors[len(deposits.Depositors)-1].Depositor
		return depositorQuery(lastAddress)
	}
	err := depositorQuery("")
	if err != nil {
		return nil, err
	}

	// for each depositor you calculate their aust balance by
	// adding together lottery_deposit /  aust_exchange_rate (the ust  denominated portion of their aust balance)
	// plus savings_aust  (the aust denominated portion of their balance)
	aUstER, err := anchor.GetAUstExchangeRate(app)
	if err != nil {
		return nil, err
	}

	snapshot := make(util.SnapshotBalanceAggregateMap)
	for _, deposit := range allDeposits {
		snapshot.AppendOrAddBalance(deposit.Depositor, util.SnapshotBalance{
			Denom:   util.DenomAUST,
			Balance: sdk.NewDecFromInt(deposit.LotteryDeposit).Quo(aUstER).TruncateInt().Add(deposit.Savings),
		})
	}
	return snapshot, nil
}
