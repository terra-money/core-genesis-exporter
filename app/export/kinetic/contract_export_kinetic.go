package kinetic

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/anchor"
	"github.com/terra-money/core/app/export/util"
	"github.com/terra-money/core/x/wasm/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	AddressKineticVault    = "terra1w93d2h57mkhkc8wgetvnj67peakvcpzgazvf2a"
	AddressKineticLockdrop = "terra140pm775hac6qzzvjd66ur44ye8llw4g5y55qw3"
	AddressKineticPhaser   = "terra1spdrct9mwsraqkm5dzrnwjc5ua90xq75t0s8p7"
	AddressKUstUstLp       = "terra16aurvlp5xctv0ftcelaseypyc89ylf4y0s5q0y"
	AddressAnchorMarket    = anchor.MoneyMarketContract
)

type cdp struct {
	Address string `json:"address"`
	Cdp     struct {
		TotalDeposited             sdk.Int `json:"total_deposited"`
		TotalDebt                  sdk.Int `json:"total_debt"`
		TotalCredit                sdk.Int `json:"total_credit"`
		LastAccumulatedYieldWeight sdk.Dec `json:"last_accumulated_yield_weight"`
	} `json:"cdp"`
}

func ExportKinetic(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Kinetic Vaults & Lockdrop")
	height := app.LastBlockHeight()
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)

	bl.RegisterAddress(util.DenomAUST, AddressKineticVault)
	bl.RegisterAddress(util.DenomUST, AddressKineticVault)

	var cdps = make([]cdp, 0)

	// loop over all cdps
	var run func(string) error
	run = func(startAfter string) error {
		var cdpsResponse struct {
			Cdps []cdp `json:"cdps"`
		}
		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: AddressKineticVault,
			QueryMsg:        []byte(fmt.Sprintf("{\"cdps\":{\"start_after\":\"%s\", \"limit\":30}}", startAfter)),
		}, &cdpsResponse); err != nil {
			return fmt.Errorf("failed to query cdps: %v", err)
		}

		cdps = append(cdps, cdpsResponse.Cdps...)

		if len(cdpsResponse.Cdps) < 30 {
			return nil
		}

		return run(cdps[len(cdps)-1].Address)
	}

	if err := run(""); err != nil {
		return nil, err
	}

	// get aUST<>UST rate
	var epochStateResponse struct {
		ExchangeRate sdk.Dec `json:"exchange_rate"`
	}
	if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: AddressAnchorMarket,
		QueryMsg:        []byte(fmt.Sprintf("{\"epoch_state\":{\"block_height\":%d}}", height)),
	}, &epochStateResponse); err != nil {
		return nil, err
	}

	var vaultBalance = make(util.SnapshotBalanceAggregateMap)
	for _, cdp := range cdps {
		// skip 0 deposit
		if cdp.Cdp.TotalDeposited.IsZero() {
			continue
		}

		vaultBalance.AppendOrAddBalance(cdp.Address, util.SnapshotBalance{
			Denom:   util.DenomAUST,
			Balance: sdk.NewDecFromInt(cdp.Cdp.TotalDeposited).Quo(epochStateResponse.ExchangeRate).TruncateInt(),
		})
	}

	lockdropSnapshot, err := exportKineticLockdrop(app, bl)
	if err != nil {
		return nil, err
	}

	finalBalance := util.MergeSnapshots(vaultBalance, lockdropSnapshot)

	return finalBalance, nil
}

func ExportKineticLpHoldings(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) (map[string]map[string]map[string]sdk.Int, error) {
	app.Logger().Info("Exporting Kinetic Lockdrop LPs")
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)
	holders, err := exportKineticLockdropShares(app)
	if err != nil {
		return nil, err
	}

	var balance struct {
		Balance sdk.Int `json:"balance"`
	}
	err = util.ContractQuery(ctx, qs, &types.QueryContractStoreRequest{
		ContractAddress: AddressKUstUstLp,
		QueryMsg:        []byte(fmt.Sprintf("{\"balance\":{\"address\": \"%s\"}}", AddressKineticLockdrop)),
	}, &balance)
	if err != nil {
		return nil, err
	}

	fmt.Printf("kinitic LP total %s\n", balance.Balance)

	totalShares := util.Sum(holders)

	for addr, b := range holders {
		holders[addr] = b.Mul(balance.Balance).Quo(totalShares)
	}

	vaultHoldings := make(map[string]map[string]map[string]sdk.Int)
	vaultHoldings[AddressKineticLockdrop] = make(map[string]map[string]sdk.Int)
	vaultHoldings[AddressKineticLockdrop][AddressKUstUstLp] = holders
	return vaultHoldings, nil
}

func exportKineticLockdropShares(app *terra.TerraApp) (map[string]sdk.Int, error) {
	ctx := util.PrepCtx(app)
	var shareHolders = make(map[string]sdk.Int)

	prefix := util.GeneratePrefix("users")
	app.WasmKeeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), util.ToAddress(AddressKineticLockdrop), prefix, func(key, value []byte) bool {
		var userInfo struct {
			UstLocked sdk.Int `json:"total_ust_locked"`
		}
		err := json.Unmarshal(value, &userInfo)
		if err != nil {
			panic(err)
		}
		shareHolders[string(key)] = userInfo.UstLocked
		return false
	})
	return shareHolders, nil
}

func exportKineticLockdrop(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)
	shareHolders, err := exportKineticLockdropShares(app)
	if err != nil {
		return nil, err
	}

	phaserBalance, err := util.GetCW20Balance(ctx, qs, util.AUST, AddressKineticPhaser)
	if err != nil {
		return nil, err
	}

	snapshot := make(util.SnapshotBalanceAggregateMap)
	totalShares := util.Sum(shareHolders)
	for addr, b := range shareHolders {
		snapshot.AppendOrAddBalance(addr, util.SnapshotBalance{
			Denom:   util.DenomAUST,
			Balance: b.Mul(phaserBalance).Quo(totalShares),
		})
	}
	return snapshot, nil
}
