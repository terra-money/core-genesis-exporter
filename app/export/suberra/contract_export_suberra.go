package app

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	wasmKeeper "github.com/terra-money/core/x/wasm/keeper"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	suberraSubwalletFactory = "terra1xmcfl8fpkq6etxznwgv58x6t7tshnjpu25a5s8"
	suberraSubwalletKey     = "accounts"
)

// ExportSuberra iterates over subwallets, then credit funds back to its owner
func ExportSuberra(app *terra.TerraApp) (map[string]util.Balance, error) {
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)

	// 1. get all suberra subwallets
	subwallets := forceIterateSubwallets(ctx, app.WasmKeeper)

	// 2. map subwallets' aUST balances
	subwalletBalances := make(map[string]sdk.Int)
	if err := iterateSubwalletsAndGetAUstBalance(ctx, qs, util.AUST, subwallets, subwalletBalances); err != nil {
		return nil, err
	}

	// 3. map subwallets to admins
	ownerBalances := make(map[string]util.Balance)
	if err := mapSubwalletToAdmin(ctx, qs, subwalletBalances, ownerBalances); err != nil {
		return nil, err
	}

	return ownerBalances, nil
}

func forceIterateSubwallets(ctx context.Context, keeper wasmKeeper.Keeper) []string {
	var subwallets []string

	prefix := util.GeneratePrefix(suberraSubwalletKey)
	addr, _ := sdk.AccAddressFromBech32(suberraSubwalletFactory)

	var address sdk.AccAddress

	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), addr, prefix, func(key, value []byte) bool {
		util.MustUnmarshalTMJSON(value, &address)
		subwallets = append(subwallets, address.String())
		return false
	})

	return subwallets
}

func iterateSubwalletsAndGetAUstBalance(ctx context.Context, q wasmtypes.QueryServer, aUST string, subwallets []string, dst map[string]sdk.Int) error {
	for _, subwallet := range subwallets {
		subwalletInString := subwallet
		bal, err := util.GetCW20Balance(ctx, q, aUST, subwalletInString)
		if err != nil {
			return err
		}

		dst[subwalletInString] = bal
	}

	return nil
}

func mapSubwalletToAdmin(ctx context.Context, q wasmtypes.QueryServer, subwalletBalances map[string]sdk.Int, ownerBalances map[string]util.Balance) error {
	var owner string
	for addr, bal := range subwalletBalances {
		if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: addr,
			QueryMsg:        []byte("{\"owner\":{}}"),
		}, &owner); err != nil {
			return err
		}

		ownerBalances[owner] = util.Balance{
			Denom:   util.DenomAUST,
			Balance: bal,
		}
	}

	return nil
}
