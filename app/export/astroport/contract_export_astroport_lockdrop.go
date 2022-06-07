package astroport

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

// This exporter covers ONLY lockdrop pools in astroport.
// Only the following Terraswap LP tokens were part of lockdrop.
//
// bLUNA/LUNA - terra1nuy34nwnsh53ygpc4xprlj263cztw7vc99leh2
// LUNA/UST - terra17dkr9rnmtmu7x4azrpupukvur2crnptyfvsrvr
// ANC/UST - terra1gecs98vcuktyfkrve9czrpgtg0m3aq586x6gzm
// MIR/UST - terra17gjf2zehfvnyjtdgua9p9ygquk6gukxe7ucgwh
// MINE/UST - terra1rqkyau9hanxtn63mjrdfhpnkpddztv3qav0tq2
// ORION/UST - terra14ffp0waxcck733a9jfd58d86h9rac2chf5xhev
// STT/UST - terra1uwhf02zuaw7grj6gjs7pxt5vuwm79y87ct5p70
// VKR/UST - terra17fysmcl52xjrs8ldswhz7n6mt37r9cmpcguack
// PSI/UST - terra1q6r8hfdl203htfvpsmyh8x689lp2g0m7856fwd
// APOLLO/UST - terra1n3gt4k3vth0uppk0urche6m3geu9eqcyujt88q
// means we only need to care about UST/LUNA/bLUNA

func ExportAstroportLockdrop(app *app.TerraApp, snapshot util.SnapshotBalanceAggregateMap) (map[string]map[string]map[string]sdk.Int, error) {

	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)
	keeper := app.WasmKeeper

	// 1. iterate all pairs & get lp token total share. key is in astroport pair
	var pm = make(poolMap)

	pairPrefix := util.GeneratePrefix("pair_info")
	factory, _ := sdk.AccAddressFromBech32(AddressAstroportFactory)
	lockdrop, _ := sdk.AccAddressFromBech32(AddressAstroportLockdrop)

	var pairAddr string
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), factory, pairPrefix, func(key, value []byte) bool {
		var p pool
		util.MustUnmarshalTMJSON(value, &pairAddr)

		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: pairAddr,
			QueryMsg:        []byte("{\"pool\":{}}"),
		}, &p); err != nil {
			panic(fmt.Errorf("unable to... %v", err))
		}

		// filter out those that have aUST/UST/LUNA
		// technically astroport doesn't have aUST pool but maybe!
		if isTargetPool(&p) {
			pm[pairAddr] = p
		}

		return false
	})

	// 2. get pools (and get astroport lp token addr then astroport pair) - key is terraswap lp address
	var liquidityPools = make(map[string]poolInfo)
	var lpLockedInGenerator = make(map[string]sdk.Int)
	poolsPrefix := util.GeneratePrefix("LiquidityPools")
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), lockdrop, poolsPrefix, func(key, value []byte) bool {
		var lpAddr = string(key)
		var pi poolInfo

		lpAddr = string(key)
		util.MustUnmarshalTMJSON(value, &pi)

		liquidityPools[lpAddr] = pi

		var lpAmount sdk.Int
		util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: AddressAstroportGenerator,
			QueryMsg:        []byte(fmt.Sprintf("{ \"deposit\": { \"lp_token\": \"%s\", \"user\": \"%s\" } }", pi.MigrationInfo.AstroportLPToken, AddressAstroportLockdrop)),
		}, &lpAmount)

		lpLockedInGenerator[pi.MigrationInfo.AstroportLPToken] = lpAmount
		return false
	})

	// 4. Iterate over all lockdrop pos
	prefix := util.GeneratePrefix("lockup_position")
	var lockupInfo struct {
		LPUnitsLocked          sdk.Int `json:"lp_units_locked"`
		AstroportLPTransferred sdk.Int `json:"astroport_lp_transferred"`
	}
	var lpShareHoldings = make(map[string]map[string]sdk.Int)

	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), lockdrop, prefix, func(key, value []byte) bool {
		terraswapLPAddress := string(key[2:46])
		userAddress := string(key[48:92])

		util.MustUnmarshalTMJSON(value, &lockupInfo)
		astroLp := liquidityPools[terraswapLPAddress].MigrationInfo.AstroportLPToken

		if lpShareHoldings[astroLp] == nil {
			lpShareHoldings[astroLp] = make(map[string]sdk.Int)
		}
		if lpShareHoldings[astroLp][userAddress].IsNil() {
			lpShareHoldings[astroLp][userAddress] = sdk.NewInt(0)
		}

		// If LP transferred is not nil, means the user has withdrawn all LPs after unlock
		if !lockupInfo.LPUnitsLocked.IsNil() && lockupInfo.AstroportLPTransferred.IsNil() {
			lpShareHoldings[astroLp][userAddress] = lpShareHoldings[astroLp][userAddress].Add(lockupInfo.LPUnitsLocked)
		}
		return false
	})

	for lp, shares := range lpShareHoldings {
		totalShares := util.Sum(shares)
		for addr, share := range shares {
			shares[addr] = share.Mul(lpLockedInGenerator[lp]).Quo(totalShares)
		}
	}

	lpContractHoldings := make(map[string]map[string]map[string]sdk.Int)
	lpContractHoldings[AddressAstroportLockdrop] = lpShareHoldings

	return lpContractHoldings, nil
}

func getShareInAssets(p pool, lpAmount sdk.Int, totalShare sdk.Int) [2]sdk.Int {
	shareRatio := sdk.ZeroDec()
	if !totalShare.IsZero() {
		shareRatio = sdk.NewDecFromInt(lpAmount).Quo(sdk.NewDecFromInt(totalShare))
	}

	return [2]sdk.Int{
		shareRatio.MulInt(p.Assets[0].Amount).TruncateInt(),
		shareRatio.MulInt(p.Assets[1].Amount).TruncateInt(),
	}
}
