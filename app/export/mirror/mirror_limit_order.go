package mirror

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"

	// wasmkeeper "github.com/terra-money/core/x/wasm/keeper"
	"github.com/terra-money/core/x/wasm/types"
	wasmtype "github.com/terra-money/core/x/wasm/types"
)

var (
	MirrorLimitOrderTokens = []string{
		util.DenomUST,
	}
	MirrorLimitOrder = "terra1zpr8tq3ts96mthcdkukmqq4y9lhw0ycevsnw89"
)

func ExportLimitOrderContract(
	app *terra.TerraApp,
	bl *util.Blacklist,
) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Mirror Limit Orders")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	orders, err := getAllOrders(ctx, q)
	if err != nil {
		return nil, err
	}
	snapshot := make(util.SnapshotBalanceAggregateMap)
	for _, order := range orders {
		for _, denom := range MirrorLimitOrderTokens {
			if order.OfferAsset.Info.Token.Addr == denom || order.OfferAsset.Info.NativeToken.Denom == denom {
				snapshot[order.Bidder] = append(snapshot[order.Bidder], util.SnapshotBalance{
					Denom:   denom,
					Balance: order.OfferAsset.Amount.Sub(order.FilledOfferAmount),
				})
			}
		}
	}

	// Blacklist resolved contracts
	for _, denom := range MirrorLimitOrderTokens {
		bl.RegisterAddress(denom, MirrorLimitOrder)
	}

	// Audit
	for _, denom := range MirrorLimitOrderTokens {
		var contractBalance sdk.Int
		if strings.Contains(denom, "terra") {
			contractBalance, err = util.GetCW20Balance(ctx, q, denom, MirrorLimitOrder)
		} else {
			contractBalance, err = util.GetNativeBalance(ctx, app.BankKeeper, denom, MirrorLimitOrder)
		}
		if err != nil {
			return snapshot, err
		}
		sumOfSnapshot := snapshot.SumOfDenom(denom)
		err = util.AlmostEqual(denom, contractBalance, sumOfSnapshot, sdk.NewInt(10000))
		if err != nil {
			return snapshot, err
		}
	}
	return snapshot, nil
}

type orderRes struct {
	Orders []order `json:"orders"`
}

type order struct {
	OrderId    int    `json:"order_id"`
	Bidder     string `json:"bidder_addr"`
	OfferAsset struct {
		Info struct {
			Token struct {
				Addr string `json:"contract_addr"`
			} `json:"token"`
			NativeToken struct {
				Denom string `json:"denom"`
			} `json:"native_token"`
		} `json:"info"`
		Amount sdk.Int `json:"amount"`
	} `json:"offer_asset"`
	FilledOfferAmount sdk.Int `json:"filled_offer_amount"`
}

func getAllOrders(ctx context.Context, q wasmtype.QueryServer) ([]order, error) {
	var getOrders func(startAfter int) error
	limit := 10
	var allOrders []order
	getOrders = func(startAfter int) error {
		var orders orderRes
		err := util.ContractQuery(ctx, q, &types.QueryContractStoreRequest{
			ContractAddress: MirrorLimitOrder,
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
