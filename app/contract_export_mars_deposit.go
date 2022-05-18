package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	marsMarket        = ""
	marsLunaLiquidity = ""
	marsUSTLiquidity  = ""
)

func ExportMarsDepositLuna(app *TerraApp, q wasmtypes.QueryServer) (map[string]sdk.Int, error) {
	ctx := prepCtx(app)
	logger := app.Logger()

	var balances = make(map[string]sdk.Int)
	logger.Info("fetching MARS liquidity (LUNA)...")

	if err := getCW20AccountsAndBalances(ctx, balances, marsLunaLiquidity, q); err != nil {
		return nil, err
	}

	// get luna liquidity <> luna er
	var lunaMarketState struct {
		LiquidityIndex sdk.Dec `json:"liquidity_index"`
	}
	if err := contractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: marsMarket,
		QueryMsg:        []byte("{\"market\": {\"asset\": {\"native\": {\"denom\": \"uluna\"}}}"),
	}, &lunaMarketState); err != nil {
		return nil, err
	}

	// balance * ER
	for address, balance := range balances {
		balances[address] = lunaMarketState.LiquidityIndex.MulInt(balance).TruncateInt()
	}

	// divide by vault size

	return balances, nil
}

func ExportMarsDepositUST(app *TerraApp, q wasmtypes.QueryServer) (map[string]sdk.Int, error) {
	ctx := prepCtx(app)
	logger := app.Logger()

	var balances = make(map[string]sdk.Int)
	logger.Info("fetching MARS liquidity (UST)...")

	if err := getCW20AccountsAndBalances(ctx, balances, marsUSTLiquidity, q); err != nil {
		return nil, err
	}

	// get luna liquidity <> luna er
	var lunaMarketState struct {
		LiquidityIndex sdk.Dec `json:"liquidity_index"`
	}
	if err := contractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: marsMarket,
		QueryMsg:        []byte("{\"market\": {\"asset\": {\"native\": {\"denom\": \"uusd\"}}}}"),
	}, &lunaMarketState); err != nil {
		return nil, err
	}

	// balance * ER
	for address, balance := range balances {
		balances[address] = lunaMarketState.LiquidityIndex.MulInt(balance).TruncateInt()
	}

	return balances, nil
}
