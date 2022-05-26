package mirror

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	"github.com/terra-money/core/x/wasm/types"
)

var (
	MirrorMint                = "terra1wfz7h3aqf4cjmjcvc6s8lxdhh7k30nkczyf0mj"
	MirrorLock                = "terra169urmlm8wcltyjsrn7gedheh7dker69ujmerv2"
	MirrorRelevantCollaterals = []string{
		util.AUST,
		util.DenomUST,
		addressLunaX,
		util.DenomLUNA,
	}
	lunaXState   = "terra1xacqx447msqp46qmv8k2sq6v5jh9fdj37az898"
	addressLunaX = "terra17y9qkl8dfkeg4py7n0g5407emqnemc3yqk5rup"
)

func ExportMirrorCdps(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	positions, err := getAllPositions(ctx, q)
	if err != nil {
		return nil, err
	}

	//fmt.Printf("mirror CPD count %d\n", len(positions))

	// get LunaX exchange rate
	lunaXExchangeRate, err := getLunaXExchangeRate(ctx, q)
	if err != nil {
		return nil, err
	}
	// fmt.Printf("got LunaX exchange rate %s\n", lunaXExchangeRate)

	snapshot := make(util.SnapshotBalanceAggregateMap)

	for _, position := range positions {
		for _, denom := range MirrorRelevantCollaterals {
			if position.Collateral.Info.Token.Addr == denom || position.Collateral.Info.NativeToken.Denom == denom {
				if position.Collateral.Info.Token.Addr == addressLunaX {
					// resolve lunaX
					lunaAmount := lunaXExchangeRate.MulInt(position.Collateral.Amount).TruncateInt()

					snapshot.AppendOrAddBalance(position.Owner, util.SnapshotBalance{Denom: util.DenomLUNA, Balance: lunaAmount})
				} else {
					// normal case (uluna, uusd, or AUST)
					snapshot.AppendOrAddBalance(position.Owner, util.SnapshotBalance{Denom: denom, Balance: position.Collateral.Amount})
				}
				break
			}
		}
	}

	// blacklist mint contract
	for _, denom := range MirrorLimitOrderTokens {
		bl.RegisterAddress(denom, MirrorMint)
	}
	// also blacklist mirror lock to prevent double incenvitizing users shorting
	bl.RegisterAddress(util.DenomUST, MirrorLock)

	return snapshot, nil
}

func Audit(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	// calculate total lunax amount
	totalLunaXAmount := sdk.ZeroInt()

	// get LunaX exchange rate
	lunaXExchangeRate, err := getLunaXExchangeRate(ctx, q)
	if err != nil {
		return err
	}

	positions, err := getAllPositions(ctx, q)
	if err != nil {
		return err
	}

	for _, position := range positions {
		if position.Collateral.Info.Token.Addr == addressLunaX {
			totalLunaXAmount = totalLunaXAmount.Add(position.Collateral.Amount)
		}
	}

	// assert everything adds up
	for _, denom := range MirrorRelevantCollaterals {
		var contractBalance sdk.Int
		if strings.Contains(denom, "terra") {
			contractBalance, err = util.GetCW20Balance(ctx, q, denom, MirrorMint)
		} else {
			contractBalance, err = util.GetNativeBalance(ctx, app.BankKeeper, denom, MirrorMint)
		}
		if err != nil {
			return err
		}

		if denom == addressLunaX {
			// compare with recorded total
			err = util.AlmostEqual(denom, contractBalance, totalLunaXAmount, sdk.NewInt(1000000))
		} else if denom == util.DenomLUNA {
			// need to include the converted lunaX
			lunaXValue := lunaXExchangeRate.MulInt(totalLunaXAmount).TruncateInt()
			expectedLunaBalance := snapshot.SumOfDenom(denom).Sub(lunaXValue)
			err = util.AlmostEqual(denom, contractBalance, expectedLunaBalance, sdk.NewInt(1000000))
		} else {
			sumOfSnapshot := snapshot.SumOfDenom(denom)
			err = util.AlmostEqual(denom, contractBalance, sumOfSnapshot, sdk.NewInt(1000000))
		}

		if err != nil {
			return err
		}
	}

	return nil
}

type positionsRes struct {
	Positions []position `json:"positions"`
}

type asset struct {
	Info struct {
		Token struct {
			Addr string `json:"contract_addr"`
		} `json:"token"`
		NativeToken struct {
			Denom string `json:"denom"`
		} `json:"native_token"`
	} `json:"info"`
	Amount sdk.Int `json:"amount"`
}

type position struct {
	Idx        sdk.Int `json:"idx"`
	Owner      string  `json:"owner"`
	Collateral asset   `json:"collateral"`
	Asset      asset   `json:"asset"`
	IsShort    bool    `json:"is_short"`
}

func getAllPositions(ctx context.Context, q types.QueryServer) ([]position, error) {
	var getPositions func(startAfter sdk.Int) error
	limit := 30
	var allPositions []position

	getPositions = func(startAfter sdk.Int) error {
		var positions positionsRes
		err := util.ContractQuery(ctx, q, &types.QueryContractStoreRequest{
			ContractAddress: MirrorMint,
			QueryMsg:        []byte(fmt.Sprintf("{\"positions\": {\"start_after\": \"%s\", \"limit\": %d, \"order_by\": \"asc\"}}", startAfter, limit)),
		}, &positions)

		if err != nil {
			return err
		}

		allPositions = append(allPositions, positions.Positions...)
		if len(positions.Positions) < limit {
			return nil
		}

		return getPositions(positions.Positions[len(positions.Positions)-1].Idx)
	}
	err := getPositions(sdk.NewInt(0))
	return allPositions, err
}

func getLunaXExchangeRate(ctx context.Context, q types.QueryServer) (sdk.Dec, error) {
	var lunaxState struct {
		State struct {
			ExchangeRate sdk.Dec `json:"exchange_rate"`
		} `json:"state"`
	}

	if err := util.ContractQuery(ctx, q, &types.QueryContractStoreRequest{
		ContractAddress: lunaXState,
		QueryMsg:        []byte("{\"state\":{}}"),
	}, &lunaxState); err != nil {
		return sdk.ZeroDec(), err
	}

	return lunaxState.State.ExchangeRate, nil
}
