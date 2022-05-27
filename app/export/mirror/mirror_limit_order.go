package mirror

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"

	"github.com/terra-money/core/x/wasm/types"
)

var (
	MirrorLimitOrder = "terra1zpr8tq3ts96mthcdkukmqq4y9lhw0ycevsnw89"
)

func ExportLimitOrderContract(
	app *terra.TerraApp,
	bl util.Blacklist,
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
		if order.OfferAsset.Info.NativeToken.Denom == util.DenomUST {
			snapshot.AppendOrAddBalance(order.Bidder, util.SnapshotBalance{Denom: util.DenomUST, Balance: order.OfferAsset.Amount.Sub(order.FilledOfferAmount)})
		}
	}

	// Blacklist resolved contracts
	bl.RegisterAddress(util.DenomUST, MirrorLimitOrder)

	return snapshot, nil
}

func AuditLOs(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	app.Logger().Info("Audit -- Mirro LO")
	ctx := util.PrepCtx(app)

	contractBalance, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomUST, MirrorLimitOrder)
	if err != nil {
		return err
	}

	sumOfSnapshot := snapshot.SumOfDenom(util.DenomUST)
	err = util.AlmostEqual(util.DenomUST, contractBalance, sumOfSnapshot, sdk.NewInt(10000))
	if err != nil {
		return err
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

func getAllOrders(ctx context.Context, q types.QueryServer) ([]order, error) {
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
