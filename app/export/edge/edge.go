package edge

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	// stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/lido"
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
		stader.StaderLunaX,
		// pLuna
		"terra1tlgelulz9pdkhls6uglfn5lmxarx7f2gxtdzh2",
	}
)

func ExportContract(app *terra.TerraApp, bl *util.Blacklist) (map[string]map[string]sdk.Int, error) {
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
		bl.RegisterAddress(market.Underlying, EdgeProtocolPool)
	}

	// header := []string{"token", "address", "amount"}
	// data := [][]string{}
	// for token, holding := range holdings {
	// 	for wallet, amount := range holding {
	// 		if !amount.IsZero() {
	// 			data = append(data, []string{token, wallet, amount.String()})
	// 		}
	// 	}
	// }
	// util.ToCsv("/home/ec2-user/edge.csv", header, data)
	return holdings, nil
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
