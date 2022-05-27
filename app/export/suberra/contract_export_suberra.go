package suberra

import (
	"context"
	"fmt"

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
func ExportSuberra(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Suberra")
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
	ownerBalances := make(util.SnapshotBalanceAggregateMap)
	if err := mapSubwalletToAdmin(ctx, qs, subwalletBalances, ownerBalances); err != nil {
		return nil, err
	}

	for _, addr := range subwallets {
		bl.RegisterAddress(util.DenomAUST, addr)
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

func mapSubwalletToAdmin(ctx context.Context, q wasmtypes.QueryServer, subwalletBalances map[string]sdk.Int, ownerBalances util.SnapshotBalanceAggregateMap) error {
	var owner string
	for addr, bal := range subwalletBalances {
		if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: addr,
			QueryMsg:        []byte("{\"owner\":{}}"),
		}, &owner); err != nil {
			return err
		}
		if bal.IsPositive() {
			ownerBalances.AppendOrAddBalance(owner,
				util.SnapshotBalance{
					Denom:   util.DenomAUST,
					Balance: bal,
				},
			)
		}
	}

	return nil
}

func Audit(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	if len(snapshot) < 2 {
		return fmt.Errorf("should have more than one sub account")
	}

	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	for owner, balance := range snapshot.FilterByDenom(util.DenomAUST) {
		var subAccount string
		err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: suberraSubwalletFactory,
			QueryMsg:        []byte(fmt.Sprintf("{ \"get_subwallet_address\": { \"owner_address\": \"%s\"}}", owner)),
		}, &subAccount)
		if err != nil {
			return err
		}
		bal, err := util.GetCW20Balance(ctx, q, util.AUST, subAccount)
		if err != nil {
			return err
		}
		util.AlmostEqual(fmt.Sprintf("suberra: %s", owner), bal, balance, sdk.NewInt(1000))
	}
	return nil
}
