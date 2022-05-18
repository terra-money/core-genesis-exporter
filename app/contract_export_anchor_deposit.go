package app

import (
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	moneyMarketContract = "terra1sepfj7s0aeg5967uxnfk4thzlerrsktkpelm5s"
	aUST                = "terra1hzh9vpxhsk8253se0vv5jj6etdvxu3nv8z07zu"
)

func ExportAnchorDeposit(app *TerraApp, q wasmtypes.QueryServer) (map[string]types.Int, error) {
	height := app.LastBlockHeight()
	ctx := app.NewContext(true, tmproto.Header{Height: height})
	newCtx := types.WrapSDKContext(ctx)
	logger := ctx.Logger()

	// scan through aUST holders, append them to accounts
	var balances = make(map[string]types.Int)
	logger.Info("fetching aUST holders...")

	if err := getCW20AccountsAndBalances(newCtx, balances, aUST, q); err != nil {
		return nil, err
	}

	// get aUST exchange rate
	var epochStateResponse struct {
		ExchangeRate string `json:"exchange_rate"`
	}
	logger.Info("fetching aUST<>UST exchange rate...")
	if err := contractQuery(newCtx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: moneyMarketContract,
		QueryMsg:        getExchangeRate(height),
	}, &epochStateResponse); err != nil {
		return nil, err
	}

	// multiply aUST exchange rate & aUST balance
	for address, bal := range balances {
		balanceInInt := types.NewDecFromInt(bal)
		erInDec, err := types.NewDecFromStr(epochStateResponse.ExchangeRate)
		if err != nil {
			panic("anchor exchange rate cannot be converted to Dec")
		}

		balances[address] = balanceInInt.Mul(erInDec).TruncateInt()
	}

	logger.Info("--- %d holders", len(balances))

	return balances, nil
}

func getAllBalancesQuery(lastAccount string) json.RawMessage {
	if lastAccount == "" {
		return []byte(fmt.Sprintf("{\"all_accounts\":{\"limit\":30}}"))
	} else {
		return []byte(fmt.Sprintf("{\"all_accounts\":{\"limit\":30,\"start_after\":\"%s\"}}", lastAccount))
	}
}

func getBalance(account string) json.RawMessage {
	return []byte(fmt.Sprintf("{\"balance\":{\"address\":\"%s\"}}", account))
}

func getExchangeRate(height int64) json.RawMessage {
	return []byte(fmt.Sprintf("{\"epoch_state\":{\"block_height\":%d}}", height))
}
