package app

import (
	"context"
	"encoding/json"
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

//
//func getPairs(ctx context.Context, q wasmtypes.QueryServer, factoryAddress string) ([]balance, error) {
//	type assetInfos []struct {
//		NativeToken struct {
//			Denom string `json:"denom"`
//		} `json:"native_token"`
//		Token struct {
//			ContractAddr string `json:"contract_addr"`
//		} `json:"token"`
//	}
//
//	type pair struct {
//		AssetInfos     assetInfos `json:"asset_infos"`
//		ContractAddr   string     `json:"contract_addr"`
//		LiquidityToken string     `json:"liquidity_token"`
//	}
//
//	type pairs struct {
//		Pairs []pair `json:"pairs"`
//	}
//
//	var iteratePairs func(startAfter assetInfos)
//	iteratePairs = func(startAfter assetInfos) pairs {
//
//	}
//
//	if err := contractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
//		ContractAddress: terraswapFactory,
//		QueryMsg:        []byte("{\"pairs\":{}}"),
//	}, &pairsResponse); err != nil {
//		return nil, err
//	}
//
//}

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
