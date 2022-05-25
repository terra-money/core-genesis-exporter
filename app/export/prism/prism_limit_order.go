package prism

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	"github.com/terra-money/core/x/wasm/types"
	wasmtype "github.com/terra-money/core/x/wasm/types"
)

var (
	PrismLimitOrderTokens = []string{
		PrismPLuna,
		PrismCLuna,
		util.DenomLUNA,
		util.DenomUST,
	}
)

func ExportLimitOrderContract(
	app *terra.TerraApp,
	snapshot util.SnapshotBalanceAggregateMap,
	bl *util.Blacklist,
) error {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	orders, err := getAllOrders(ctx, q)
	if err != nil {
		return err
	}
	holdings := make(map[string]map[string]sdk.Int)
	for _, order := range orders {
		for _, denom := range PrismLimitOrderTokens {
			if order.OfferAsset.Info.Cw20 == denom || order.OfferAsset.Info.Native == denom {
				if holdings[denom] == nil {
					holdings[denom] = make(map[string]sdk.Int)
				}
				if holdings[denom][order.Bidder].IsNil() {
					holdings[denom][order.Bidder] = order.OfferAsset.Amount
				} else {
					holdings[denom][order.Bidder] = holdings[denom][order.Bidder].Add(order.OfferAsset.Amount)
				}
			}
		}
	}

	// Audit
	for _, denom := range PrismLimitOrderTokens {
		bl.RegisterAddress(util.MapContractToDenom(denom), PrismLimitOrder)
		var contractBalance sdk.Int
		if strings.Contains(denom, "terra") {
			contractBalance, err = util.GetCW20Balance(ctx, q, denom, PrismLimitOrder)
		} else {
			contractBalance, err = util.GetNativeBalance(ctx, app.BankKeeper, denom, PrismLimitOrder)
		}
		if err != nil {
			return err
		}
		err = util.AlmostEqual(denom, contractBalance, util.Sum(holdings[denom]), sdk.NewInt(10000))
		if err != nil {
			return err
		}
		snapshot.Add(holdings[denom], util.MapContractToDenom(denom))
	}
	return nil
}

type orderRes struct {
	Orders []order `json:"orders"`
}

type order struct {
	OrderId    int    `json:"order_id"`
	Bidder     string `json:"bidder_addr"`
	OfferAsset struct {
		Info struct {
			Native string `json:"native"`
			Cw20   string `json:"cw20"`
		}
		Amount sdk.Int `json:"amount"`
	} `json:"offer_asset"`
}

func getAllOrders(ctx context.Context, q wasmtype.QueryServer) ([]order, error) {
	var getOrders func(startAfter int) error
	limit := 10
	var allOrders []order
	getOrders = func(startAfter int) error {
		var orders orderRes
		err := util.ContractQuery(ctx, q, &types.QueryContractStoreRequest{
			ContractAddress: PrismLimitOrder,
			QueryMsg:        []byte(fmt.Sprintf("{\"orders\": {\"start_after\": %d, \"limit\": %d, \"order_by\": \"asc\"}}", startAfter, limit)),
		}, &orders)

		if err != nil {
			return err
		}
		allOrders = append(allOrders, orders.Orders...)
		if len(orders.Orders) < limit {
			return nil
		}
		return getOrders(orders.Orders[len(orders.Orders)-1].OrderId)
	}
	err := getOrders(0)
	return allOrders, err
}
