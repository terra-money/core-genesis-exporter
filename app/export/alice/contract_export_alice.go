package alice

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app"
	wasmKeeper "github.com/terra-money/core/x/wasm/keeper"
)

var (
	AliceaaUSTWrapper = "terra14el5cs9v3ezu57fc32kx2ltad4m3elg3le8twp"
)

// ExportAlice iterates over aaUST owners & extract balance
// 1aaUST = 1aUST
func ExportAlice(terra *app.TerraApp, b Blacklist) (map[string]Balance, error) {
	// register blacklist
	b.RegisterAddress(DenomAUST, AliceaaUSTWrapper)

	// iterate through cw20 users (force iterate since alice contract doesn't implement all_accounts)
	// then get balance
	// 1aaUST = 1aUST
	ctx := PrepCtx(terra)
	var balances = make(map[string]Balance)
	if err := forceIterateAndFindWalletAndBalance(ctx, terra.WasmKeeper, AliceaaUSTWrapper, balances); err != nil {
		return nil, err
	}

	return balances, nil
}

func forceIterateAndFindWalletAndBalance(ctx context.Context, keeper wasmKeeper.Keeper, aaUST string, balances map[string]balance) error {
	prefix := GeneratePrefix("balance")
	addr, _ := sdk.AccAddressFromBech32(aaUST)

	var bal string
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), addr, prefix, func(key, value []byte) bool {
		MustUnmarshalTMJSON(value, &bal)

		balInInt, _ := sdk.NewIntFromString(bal)
		balances[string(key)] = Balance{
			Denom:   DenomAUST,
			Balance: balInInt,
		}

		return false
	})

	return nil
}
