package mars

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	marsMarket  = "terra19dtgj9j5j7kyf3pmejqv8vzfpxtejaypgzkz5u"
	maLunaToken = "terra1x4rrkxx5pyuce32wsdn8ypqnpx8n27klnegv0d"
	maUstToken  = "terra1cuku0vggplpgfxegdrenp302km26symjk4xxaf"
	// TODO: assign safety fund to mars multisig
	marsSafetyFund = "terra16zrcxq6pyq7uxhcmgfe68p09xh6g4wk6yw2f70"
	marsFields     = []string{
		//marsLunaUstField
		"terra1kztywx50wv38r58unxj9p6k3pgr2ux6w5x68md",
		// marsAncUstField
		"terra1vapq79y9cqghqny7zt72g4qukndz282uvqwtz6",
		// marsMirUstField
		"terra12dq4wmfcsnz6ycep6ek4umtuaj6luhfp256hyu",
	}
	astroportGenerator = "terra1zgrx9jjqrfye8swykfgmd6hpde60j0nszzupp9"
)

// To prevent double counting, snapshot only assign depositors what is left in the 'bank'
// borrowers are eligible for the snapshot
// Logic:
// 1. Find ownership of maTokens
// 2. Find total supply of maTokens
// 3. Find balance of assets in bank
// 4. Assign accounts with assets proportionally
func ExportContract(app *terra.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	lunaSs, err := ExportMarsDepositLuna(app, bl)
	if err != nil {
		return nil, err
	}
	ustSs, err := ExportMarsDepositUST(app, bl)
	if err != nil {
		return nil, err
	}
	safetySs, err := ExportMarsSafetyFund(app, bl)
	if err != nil {
		return nil, err
	}
	marsSs := util.MergeSnapshots(lunaSs, ustSs, safetySs)
	return marsSs, nil
}

func Audit(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	ctx := util.PrepCtx(app)

	// UST
	ustLockedInBank, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomUST, marsMarket)
	if err != nil {
		return err
	}
	ustLockedInSafety, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomUST, marsSafetyFund)
	if err != nil {
		return err
	}

	err = util.AlmostEqual("mars uust audit", ustLockedInBank.Add(ustLockedInSafety), snapshot.SumOfDenom(util.DenomUST), sdk.NewInt(1000000))
	if err != nil {
		return err
	}

	// Luna
	lunaLockedInBank, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomLUNA, marsMarket)
	if err != nil {
		return err
	}
	err = util.AlmostEqual("mars uluna audit", ustLockedInBank.Add(lunaLockedInBank), snapshot.SumOfDenom(util.DenomLUNA), sdk.NewInt(1000000))
	if err != nil {
		return err
	}
	return nil
}

func ExportMarsDepositLuna(app *terra.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	logger := app.Logger()

	var balances = make(map[string]sdk.Int)
	logger.Info("fetching MARS liquidity (LUNA)...")

	if err := util.GetCW20AccountsAndBalances2(ctx, app.WasmKeeper, maLunaToken, balances); err != nil {
		return nil, err
	}

	marsLunaBalance, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomLUNA, marsMarket)
	if err != nil {
		return nil, err
	}
	totalSupply, err := util.GetCW20TotalSupply(ctx, q, maLunaToken)
	if err != nil {
		return nil, err
	}

	sum := sdk.NewInt(0)
	// balance * ER
	for address, balance := range balances {
		if balance.IsZero() {
			continue
		}
		balances[address] = balance.Mul(marsLunaBalance).Quo(totalSupply)
		sum = sum.Add(balances[address])
	}

	snapshot := make(util.SnapshotBalanceAggregateMap)
	// Black listing Mars Market Contract for deduplication later
	bl.RegisterAddress(util.DenomLUNA, marsMarket)
	snapshot.Add(balances, util.DenomLUNA)
	return snapshot, nil
}

func ExportMarsDepositUST(app *terra.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	logger := app.Logger()

	var balances = make(util.BalanceMap)
	logger.Info("fetching MARS liquidity (UST)...")

	if err := util.GetCW20AccountsAndBalances(ctx, app.WasmKeeper, maUstToken, balances); err != nil {
		return nil, err
	}

	// get luna liquidity <> luna er
	var lunaMarketState struct {
		LiquidityIndex sdk.Dec `json:"liquidity_index"`
	}
	if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: marsMarket,
		QueryMsg:        []byte("{\"market\": {\"asset\": {\"native\": {\"denom\": \"uusd\"}}}}"),
	}, &lunaMarketState); err != nil {
		return nil, err
	}

	// balance * ER
	for address, balance := range balances {
		balances[address] = lunaMarketState.LiquidityIndex.MulInt(balance).TruncateInt()
	}

	snapshot := make(util.SnapshotBalanceAggregateMap)
	// Black listing Mars Market Contract for deduplication later
	bl.RegisterAddress(util.DenomUST, marsMarket)
	snapshot.Add(balances, util.DenomUST)
	return snapshot, nil
}

func ExportMarsSafetyFund(app *terra.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	balance, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomUST, marsSafetyFund)
	if err != nil {
		return nil, err
	}
	info, err := app.WasmKeeper.GetContractInfo(sdk.UnwrapSDKContext(ctx), sdk.AccAddress(marsSafetyFund))
	if err != nil {
		return nil, err
	}
	snapshot := make(util.SnapshotBalanceAggregateMap)
	snapshot[info.Admin] = append(snapshot[info.Admin], util.SnapshotBalance{
		Denom:   util.DenomUST,
		Balance: balance,
	})
	bl.RegisterAddress(util.DenomUST, marsSafetyFund)
	return snapshot, nil
}

// Get eventual ownership of LP tokens in the Field of Mars (leveraged yield farming) contracts
// 1. Get the LP token contract addr
// 2. List all positions recurrsively
// 3. Find how much LP tokens are deposited at the astroport generator
// 4. Split the LP based on bond_unit and create a holding map with format {farm: {"lp_token_addr": {"wallet_addr": "amount"}}}
func ExportFieldOfMarsLpTokens(app *terra.TerraApp) (map[string]map[string]map[string]sdk.Int, error) {
	app.Logger().Info("Exporting Field of Mars")
	q := util.PrepWasmQueryServer(app)
	ctx := util.PrepCtx(app)
	holdings := make(map[string]map[string]map[string]sdk.Int)
	lpTokenFieldMap := make(map[string]string)
	for _, fieldContract := range marsFields {
		holding := make(map[string]map[string]sdk.Int)
		err := getFieldOfMarsPositions(ctx, q, fieldContract, holding, lpTokenFieldMap)
		holdings[fieldContract] = holding
		if err != nil {
			app.Logger().Error(err.Error())
			return nil, err
		}
	}

	for _, holding := range holdings {
		for lpToken, h := range holding {
			fieldContract := lpTokenFieldMap[lpToken]
			err := auditAstroportLpBalances(ctx, q, astroportGenerator, lpToken, h, fieldContract)
			if err != nil {
				return nil, err
			}

		}
	}
	return holdings, nil
}

func auditAstroportLpBalances(ctx context.Context, q wasmtypes.QueryServer, astroportGenerator string, lpToken string, holdings map[string]sdk.Int, vaultAddr string) error {
	astroportDeposits, err := getAstroportGeneratorDeposit(ctx, q, astroportGenerator, lpToken, vaultAddr)
	if err != nil {
		return err
	}
	totalHolding := sdk.NewInt(0)
	for _, balance := range holdings {
		totalHolding = totalHolding.Add(balance)
	}

	err = util.AlmostEqual("mars farm", totalHolding, astroportDeposits, sdk.NewInt(100000))
	if err != nil {
		return err
	}
	return nil
}

func getAstroportGeneratorDeposit(ctx context.Context, q wasmtypes.QueryServer, astroportGenerator string, lpToken string, user string) (sdk.Int, error) {
	query := fmt.Sprintf("{\"deposit\": {\"user\": \"%s\", \"lp_token\": \"%s\"}}", user, lpToken)
	var amount sdk.Int
	err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: astroportGenerator,
		QueryMsg:        []byte(query),
	}, &amount)
	if err != nil {
		return amount, err
	}
	return amount, nil
}

func getFieldOfMarsPositions(
	ctx context.Context,
	q wasmtypes.QueryServer,
	fieldContract string,
	holdings map[string]map[string]sdk.Int,
	lpTokenFieldMap map[string]string,
) error {
	var fieldConfig struct {
		PrimaryPair struct {
			LiquidityToken string `json:"liquidity_token"`
		} `json:"primary_pair"`
	}
	err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: fieldContract,
		QueryMsg:        []byte("{\"config\":{}}"),
	}, &fieldConfig)
	if err != nil {
		return err
	}
	lpTokenFieldMap[fieldConfig.PrimaryPair.LiquidityToken] = fieldContract

	var fieldState struct {
		TotalBondUnits sdk.Int `json:"total_bond_units"`
	}
	err = util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: fieldContract,
		QueryMsg:        []byte("{\"state\": {}}"),
	}, &fieldState)
	if err != nil {
		return err
	}

	astroportGeneratorBalance, err := getAstroportGeneratorDeposit(ctx, q, astroportGenerator, fieldConfig.PrimaryPair.LiquidityToken, fieldContract)
	if err != nil {
		return err
	}

	type Position struct {
		User     string `json:"user"`
		Position struct {
			BondUnits sdk.Int `json:"bond_units"`
		} `json:"position"`
	}

	limit := 20
	var positions []Position
	var getPositions func(string) error
	getPositions = func(lastAcc string) error {
		// fmt.Printf("last account: %s, len: %d\n", lastAcc, len(positions))
		query := fmt.Sprintf("{\"positions\":{\"limit\": %d,\"start_after\":\"%s\"}}", limit, lastAcc)
		var pagedPositions []Position
		err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: fieldContract,
			QueryMsg:        []byte(query),
		}, &pagedPositions)
		if err != nil {
			return err
		}
		positions = append(positions, pagedPositions...)
		if len(pagedPositions) < limit {
			return nil
		} else {
			return getPositions(pagedPositions[len(pagedPositions)-1].User)
		}
	}
	err = getPositions("")
	if err != nil {
		return err
	}
	// fmt.Printf("number of positions: %d\n", len(positions))

	lpHoldings := make(map[string]sdk.Int)
	for _, pos := range positions {
		lpHoldings[pos.User] = pos.Position.BondUnits.Mul(astroportGeneratorBalance).Quo(fieldState.TotalBondUnits)
	}
	holdings[fieldConfig.PrimaryPair.LiquidityToken] = lpHoldings
	return nil
}
