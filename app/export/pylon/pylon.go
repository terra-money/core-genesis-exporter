package pylon

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	// stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"

	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	PylonPools = []string{
		// Mine
		"terra1z5j60wct88yz62ylqa4t8p8239cwx9kjlghkg2",
		// Loop
		"terra149fxy4crxnhy4z2lezefwe7evjthlsttyse20m",
		// TWD
		"terra1he8j44cv2fcntujjnsqn3ummauua555agz4js0",
		// PSI
		"terra1xu84jh7x2ugt3gkpv8d450hdwcyejtcwwkkzgv",
		// VKR
		"terra1zxtcxxjqp7c46g8jx0t25s5ysa5qawmwd2w7nr",
		// GP
		"terra1jk0xh49ft2ls4u9dlfqweed8080u6ysumvmtcz",
		// ORION
		"terra15y9r79wlu8uqvlu3k7vgv0kgdy29m8j9tt9xgg",
		// Deviants Faction
		"terra19zn5u7ej083em99was4y02j3yhracnxwxcvmt4",
		// Whale
		"terra15y70slq4l4s5da2etsyqasyjht0dnquj03qm05",
		// Glow
		"terra1g9kzlt58ycppx9elymnrgxmwssfawym668r2y4",
		// SAYVE
		"terra1he8ymhmqmtuu5699akwk94urap6z09xnnews32",
		// XDEFI
		"terra1vftcl08p73v3nkuwvv5ntznku44s7p2tq00mgn",
		// Luna Bulls
		"terra132u62nsympysvtg3nng5xg6tjf6cr8sxrq7ena",
		// TerraBots
		"terra1dyattlzq58ty7pat337a9dz6j46thldu5gn8ls",
		// Lunart
		"terra1xkw8vusucy9c2w9hxuw6lktxk2w8g72utdyq96",
	}
)

type PylonPoolConfig struct {
	Pool       string
	AUstAmount sdk.Int
	UstAmount  sdk.Int
	PoolToken  string `json:"dp_token"`
}

func ExportContract(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Pylon")
	var _ wasmtypes.QueryServer
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	snapshot := make(util.SnapshotBalanceAggregateMap)

	for _, pool := range PylonPools {
		config, err := getConfig(ctx, q, app.BankKeeper, pool)
		if err != nil {
			return nil, err
		}
		totalSupply, err := util.GetCW20TotalSupply(ctx, q, config.PoolToken)
		if err != nil {
			return nil, err
		}
		tokenBalances := make(map[string]sdk.Int)
		err = util.GetCW20AccountsAndBalances2(ctx, app.WasmKeeper, config.PoolToken, tokenBalances)
		if err != nil {
			return nil, err
		}

		for w, a := range tokenBalances {
			userUstAmount := a.Mul(config.UstAmount).Quo(totalSupply)
			userAustAmount := a.Mul(config.AUstAmount).Quo(totalSupply)

			if !userUstAmount.IsZero() {
				snapshot.AppendOrAddBalance(w, util.SnapshotBalance{Denom: util.DenomUST, Balance: userUstAmount})
			}
			if !userAustAmount.IsZero() {
				snapshot.AppendOrAddBalance(w, util.SnapshotBalance{Denom: util.DenomAUST, Balance: userAustAmount})
			}
		}
	}

	return snapshot, nil
}

func Audit(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	app.Logger().Info("Audit -- Pylon")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	sumAUst := sdk.NewInt(0)
	sumUst := sdk.NewInt(0)

	for _, pool := range PylonPools {
		config, err := getConfig(ctx, q, app.BankKeeper, pool)
		if err != nil {
			return err
		}
		sumAUst = sumAUst.Add(config.AUstAmount)
		sumUst = sumUst.Add(config.UstAmount)
	}

	if err := util.AlmostEqual(util.AUST, sumAUst, snapshot.SumOfDenom(util.DenomAUST), sdk.NewInt(1000000)); err != nil {
		return err
	}
	if err := util.AlmostEqual(util.DenomUST, sumUst, snapshot.SumOfDenom(util.DenomUST), sdk.NewInt(1000000)); err != nil {
		return err
	}

	return nil
}

func getConfig(ctx context.Context, q wasmtypes.QueryServer, k wasmtypes.BankKeeper, pool string) (PylonPoolConfig, error) {
	var config PylonPoolConfig
	err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: pool,
		QueryMsg:        []byte("{\"config\":{}}"),
	}, &config)
	config.Pool = pool
	if err != nil {
		return config, err
	}
	aUstBalance, err := util.GetCW20Balance(ctx, q, util.AUST, pool)
	if err != nil {
		return config, err
	}
	config.AUstAmount = aUstBalance
	ustBalance, err := util.GetNativeBalance(ctx, k, util.DenomUST, pool)
	if err != nil {
		return config, err
	}
	config.UstAmount = ustBalance
	return config, err
}
