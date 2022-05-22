package app

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	wasmKeeper "github.com/terra-money/core/x/wasm/keeper"
)

var (
	AliceaaUSTWrapper = "terra14el5cs9v3ezu57fc32kx2ltad4m3elg3le8twp"
)

// ExportAlice iterates over aaUST owners & extract balance
// 1aaUST = 1aUST
func ExportAlice(app *TerraApp, b blacklist) (map[string]balance, error) {
	// register blacklist
	b.RegisterAddress(DenomAUST, AliceaaUSTWrapper)

	// iterate through cw20 users (force iterate since alice contract doesn't implement all_accounts)
	// then get balance
	// 1aaUST = 1aUST
	ctx := prepCtx(app)
	var balances = make(map[string]balance)
	if err := forceIterateAndFindWalletAndBalance(ctx, app.WasmKeeper, AliceaaUSTWrapper, balances); err != nil {
		return nil, err
	}

	return balances, nil
}

func forceIterateAndFindWalletAndBalance(ctx context.Context, keeper wasmKeeper.Keeper, aaUST string, balances map[string]balance) error {
	prefix := generatePrefix("balance")
	addr, _ := sdk.AccAddressFromBech32(aaUST)

	var bal string
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), addr, prefix, func(key, value []byte) bool {
		mustUnmarshalTMJSON(value, &bal)

		balInInt, _ := sdk.NewIntFromString(bal)
		balances[string(key)] = balance{
			Denom:   DenomAUST,
			Balance: balInInt,
		}

		return false
	})

	return nil
}
