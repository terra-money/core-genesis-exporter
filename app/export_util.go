package app

import (
	"context"
	"encoding/json"
	"github.com/cosmos/cosmos-sdk/store"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

type allAccountsResponse struct {
	Accounts []string `json:"accounts"`
}

type balanceResponse struct {
	Balance sdktypes.Int `json:"balance"`
}

type balance struct {
	Denom   string       `json:"denom"`
	Balance sdktypes.Int `json:"balance"`
}

func prepCtx(app *TerraApp) context.Context {
	height := app.LastBlockHeight()
	ctx := app.NewContext(true, tmproto.Header{Height: height})
	return sdktypes.WrapSDKContext(ctx)
}

func getCW20AccountsAndBalances(ctx context.Context, balanceMap map[string]sdktypes.Int, contractAddress string, q wasmtypes.QueryServer) error {
	var allAccounts allAccountsResponse
	var accounts []string

	var getAccounts func(lastAccount string) error
	getAccounts = func(lastAccount string) error {
		// get aUST balance
		// lcd.terra.dev/wasm/contracts/terra1..../store?query_msg={"balance":{"address":"terra1...."}}
		response, err := q.ContractStore(ctx, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: contractAddress,
			QueryMsg:        getAllBalancesQuery(lastAccount),
		})

		if err != nil {
			return err
		}

		unmarshalErr := tmjson.Unmarshal(response.QueryResult, &allAccounts)
		if unmarshalErr != nil {
			return unmarshalErr
		}

		accounts = append(accounts, allAccounts.Accounts...)

		if len(allAccounts.Accounts) < 30 {
			return nil
		} else {
			return getAccounts(allAccounts.Accounts[len(allAccounts.Accounts)-1])
		}
	}

	if err := getAccounts(""); err != nil {
		return err
	}

	// now accounts slice is filled, get actual balances
	var balance balanceResponse
	var getAnchorUSTBalance func(account string) error
	getAnchorUSTBalance = func(account string) error {
		response, err := q.ContractStore(ctx, &wasmtypes.QueryContractStoreRequest{
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

		balanceMap[account] = balance.Balance

		return nil
	}

	for _, account := range accounts {
		if err := getAnchorUSTBalance(account); err != nil {
			return err
		}
	}

	return nil
}

func contractQuery(ctx context.Context, q wasmtypes.QueryServer, req *wasmtypes.QueryContractStoreRequest, res interface{}) error {
	response, err := q.ContractStore(ctx, req)
	if err != nil {
		return err
	}

	unmarshalErr := json.Unmarshal(response.QueryResult, res)
	if unmarshalErr != nil {
		return unmarshalErr
	}

	return nil
}

// calculateIteratorStartKey calculates start key for an iterator -- useful in case where specific querier is not
// available from within the contract itself (i.e. LP stakers list from staking contract)
func calculateIteratorStartKey(store store.KVStore, ctx context.Context, q wasmtypes.QueryServer, contractAddress string, prefix []byte) ([]byte, error) {
	return nil, nil
}
