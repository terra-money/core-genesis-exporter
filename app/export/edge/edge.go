package edge

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	// stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/lido"
	"github.com/terra-money/core/app/export/prism"
	"github.com/terra-money/core/app/export/stader"
	util "github.com/terra-money/core/app/export/util"

	// wasmkeeper "github.com/terra-money/core/x/wasm/keeper"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	EdgeProtocolPool   = "terra1pcxwtrxppj9xj7pq3k95wm2zztfr9kwfkcgq0w"
	EdgeProtocolTokens = []string{
		util.AUST,
		util.DenomLUNA,
		util.DenomLUNA,
		lido.BLuna,
		lido.StLuna,
		stader.LunaX,
		// pLuna
		prism.PrismPLuna,
	}
)

func ExportContract(app *terra.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Edge Protocol")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	markets, err := getMarkets(ctx, q)
	if err != nil {
		return nil, err
	}
	holdings := make(map[string]map[string]sdk.Int)
	for _, market := range markets {
		// Skip not whitelisted tokens
		if !contains(EdgeProtocolTokens, market.Underlying) {
			continue
		}
		totalSupply, err := util.GetCW20TotalSupply(ctx, q, market.Etoken)
		if err != nil {
			return nil, err
		}
		accountBalances := make(map[string]sdk.Int)
		err = util.GetCW20AccountsAndBalances2(ctx, app.WasmKeeper, market.Etoken, accountBalances)
		if err != nil {
			return nil, err
		}
		holding := make(map[string]sdk.Int)
		for k, v := range accountBalances {
			// Insurance fund belongs to the protocol
			holding[k] = v.Mul(market.TotalAmount.Sub(market.InsuranceAmount.TruncateInt())).Quo(totalSupply)
		}

		// Assigning insurance fund to protocol admin
		poolAddr, err := sdk.AccAddressFromBech32(EdgeProtocolPool)
		if err != nil {
			return nil, err
		}
		info, err := app.WasmKeeper.GetContractInfo(sdk.UnwrapSDKContext(ctx), poolAddr)
		if err != nil {
			return nil, err
		}
		holding[info.Admin] = market.InsuranceAmount.TruncateInt()
		holdings[market.Underlying] = holding
		bl.RegisterAddress(util.MapContractToDenom(market.Underlying), EdgeProtocolPool)
	}

	snapshot := make(util.SnapshotBalanceAggregateMap)
	for asset, holding := range holdings {
		for addr, b := range holding {
			snapshot[addr] = append(snapshot[addr], util.SnapshotBalance{
				Denom:   asset,
				Balance: b,
			})
		}
	}
	return snapshot, nil
}

func Audit(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	for _, token := range EdgeProtocolTokens {
		var balance sdk.Int
		var err error
		if strings.Contains(token, "terra") {
			balance, err = util.GetCW20Balance(ctx, q, token, EdgeProtocolPool)
			if err != nil {
				return err
			}
		} else {
			balance, err = util.GetNativeBalance(ctx, app.BankKeeper, token, EdgeProtocolPool)
			if err != nil {
				return err
			}
		}
		denom := util.MapContractToDenom(token)
		err = util.AlmostEqual(fmt.Sprintf("edge: %s", denom), snapshot.SumOfDenom(denom), balance, sdk.NewInt(100000))
		if err != nil {
			return err
		}
	}
	return nil
}

func contains(a []string, i string) bool {
	for _, e := range a {
		if e == i {
			return true
		}
	}
	return false
}

func getAllAccounts(ctx context.Context, q wasmtypes.QueryServer) ([]string, error) {
	var getAccounts func(startAfter string) error
	limit := 20
	var allAccounts []string
	getAccounts = func(startAfter string) error {
		var accounts []string
		err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: EdgeProtocolPool,
			QueryMsg:        []byte(fmt.Sprintf("{ \"batch\": { \"all_accounts\": { \"start_after\": \"%s\", \"limit\": %d } } }", startAfter, limit)),
		}, &accounts)
		if err != nil {
			return err
		}
		allAccounts = append(allAccounts, accounts...)
		if len(accounts) < 20 {
			return nil
		}
		return getAccounts(accounts[len(accounts)-1])
	}
	err := getAccounts("")
	if err != nil {
		return nil, err
	}
	return allAccounts, nil
}

type Market struct {
	Underlying      string  `json:"underlying"`
	Etoken          string  `json:"etoken_addr"`
	TotalAmount     sdk.Int `json:"total_credit"`
	InsuranceAmount sdk.Dec `json:"total_insurance"`
}

func getMarkets(ctx context.Context, q wasmtypes.QueryServer) ([]Market, error) {
	var markets []Market
	err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: EdgeProtocolPool,
		QueryMsg:        []byte("{\"market_lists\":{}}"),
	}, &markets)
	if err != nil {
		return nil, err
	}
	return markets, err
}
