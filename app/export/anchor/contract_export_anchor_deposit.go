package anchor

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

func ExportAnchorDeposit(terra *app.TerraApp, q wasmtypes.QueryServer) (map[string]types.Int, error) {
	height := terra.LastBlockHeight()
	ctx := terra.NewContext(true, tmproto.Header{Height: height})
	newCtx := types.WrapSDKContext(ctx)
	logger := ctx.Logger()

	// scan through aUST holders, append them to accounts
	var balances = make(map[string]types.Int)
	logger.Info("fetching aUST holders...")

	if err := util.GetCW20AccountsAndBalances(newCtx, balances, AUST, q); err != nil {
		return nil, err
	}

	// get aUST exchange rate
	var epochStateResponse struct {
		ExchangeRate string `json:"exchange_rate"`
	}
	logger.Info("fetching aUST<>UST exchange rate...")
	if err := util.ContractQuery(newCtx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: MoneyMarketContract,
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

func getExchangeRate(height int64) json.RawMessage {
	return []byte(fmt.Sprintf("{\"epoch_state\":{\"block_height\":%d}}", height))
}
