package app

import (
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	moneyMarketContract = "terra1sepfj7s0aeg5967uxnfk4thzlerrsktkpelm5s"
	aUST                = "terra1hzh9vpxhsk8253se0vv5jj6etdvxu3nv8z07zu"
)

type allAccountsResponse struct {
	Accounts []string `json:"accounts"`
}

type balanceResponse struct {
	Balance string `json:"balance"`
}

func ExportAnchorDeposit(ctx types.Context, ms types.CommitMultiStore, height int64, q wasmtypes.QueryServer) (map[string]string, error) {
	newCtx := types.WrapSDKContext(ctx.WithMultiStore(ms))
	logger := ctx.Logger()

	// scan through aUST holders, append them to accounts
	var allAccounts allAccountsResponse
	var accounts []string
	var balances = make(map[string]string)
	var getAnchorUSTAccounts func(lastAccount string) error
	logger.Info("fetching aUST holders...")

	getAnchorUSTAccounts = func(lastAccount string) error {
		// get aUST balance
		response, err := q.ContractStore(newCtx, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: aUST,
			QueryMsg:        getAllBalancesQuery(lastAccount),
		})

		if err != nil {
			return err
		}

		unmarshalErr := json.Unmarshal(response.QueryResult, &allAccounts)
		if unmarshalErr != nil {
			return unmarshalErr
		}

		accounts = append(accounts, allAccounts.Accounts...)

		if len(allAccounts.Accounts) < 30 {
			return nil
		} else {
			return getAnchorUSTAccounts(allAccounts.Accounts[len(allAccounts.Accounts)-1])
		}
	}

	if err := getAnchorUSTAccounts(""); err != nil {
		return nil, err
	}

	// now accounts slice is filled, get actual balances
	var balance balanceResponse
	var getAnchorUSTBalance func(account string) error
	logger.Info("fetching aUST balances...")
	getAnchorUSTBalance = func(account string) error {
		response, err := q.ContractStore(newCtx, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: aUST,
			QueryMsg:        getBalance(account),
		})

		if err != nil {
			return err
		}

		unmarshalErr := json.Unmarshal(response.QueryResult, &balance)
		if unmarshalErr != nil {
			return unmarshalErr
		}

		balances[account] = balance.Balance

		return nil
	}

	for _, account := range accounts {
		if err := getAnchorUSTBalance(account); err != nil {
			return nil, err
		}
	}

	// get aUST exchange rate
	var epochStateResponse struct {
		ExchangeRate string `json:"exchange_rate"`
	}
	logger.Info("fetching aUST<>UST exchange rate...")
	response, err := q.ContractStore(newCtx, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: moneyMarketContract,
		QueryMsg:        getExchangeRate(height),
	})
	if err != nil {
		return nil, err
	}

	unmarshalErr := json.Unmarshal(response.QueryResult, &epochStateResponse)
	if unmarshalErr != nil {
		return nil, unmarshalErr
	}

	// multiply aUST exchange rate & aUST balance
	for address, bal := range balances {
		balanceInInt, err := types.NewDecFromStr(bal)
		if err != nil {
			panic("balance cannot be converted to Dec")
		}

		erInDec, err := types.NewDecFromStr(epochStateResponse.ExchangeRate)
		if err != nil {
			panic("anchor exchange rate cannot be converted to Dec")
		}

		balances[address] = balanceInInt.Mul(erInDec).TruncateInt().String()
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
