package steak

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	AddressSteakHub   = "terra15qr8ev2c0a0jswjtfrhfaj5ucgkhjd7la2shlg"
	AddressSteakToken = "terra1rl4zyexjphwgx6v3ytyljkkc4mrje2pyznaclv"
)

func ExportSteak(app *app.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceMap, error) {
	// Blacklist steak hub from LUNA balance snapshot
	bl.RegisterAddress(util.DenomLUNA, AddressSteakHub)

	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)

	// 1. Iterate over all Steak token holders, get their balance
	var balanceMap = make(map[string]sdk.Int)
	if err := util.GetCW20AccountsAndBalances(ctx, app.WasmKeeper, AddressSteakToken, balanceMap); err != nil {
		return nil, fmt.Errorf("error during cw20 iteration: %v", err)
	}

	// 2. Get Steak<>LUNA Exchange Rate
	var hubState struct {
		ExchangeRate sdk.Dec `json:"exchange_rate"`
	}
	if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: AddressSteakHub,
		QueryMsg:        []byte("{\"state\":{}}"),
	}, &hubState); err != nil {
		return nil, fmt.Errorf("failed to query SteakHub state: %v", err)
	}

	// 3. Iterate over balanceMap and apply exchange rate
	var finalBalance = make(util.SnapshotBalanceMap)
	for addr, bal := range balanceMap {
		finalBalance[addr] = util.SnapshotBalance{
			Denom:   util.DenomLUNA,
			Balance: hubState.ExchangeRate.MulInt(bal).TruncateInt(),
		}
	}

	return finalBalance, nil
}
