package astroport

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	AddressAstroportLockdrop = "terra1627ldjvxatt54ydd3ns6xaxtd68a2vtyu7kakj"
	AddressAstroportFactory  = "terra1fnywlw4edny3vw44x04xd67uzkdqluymgreu7g"
	AddressAUST              = "terra1hzh9vpxhsk8253se0vv5jj6etdvxu3nv8z07zu"
	AddressBLUNA             = "terra1kc87mu460fwkqte29rquh4hc20m54fxwtsx7gp"
)

type (
	pool struct {
		Asset [2]struct {
			AssetInfo assetInfo `json:"info"`
			Amount    sdk.Int   `json:"amount"`
		} `json:"assets"`
		TotalShare sdk.Int `json:"total_share"`
	}

	// only care about migration info to figure out ts -> astro migrated
	poolInfo struct {
		MigrationInfo struct {
			AstroportLPToken string `json:"astroport_lp_token"`
		} `json:"migration_info"`
	}

	assetInfo struct {
		Token *struct {
			ContractAddr string `json:"contract_addr"`
		} `json:"token,omitempty"`
		NativeToken *struct {
			Denom string `json:"denom"`
		} `json:"native_token,omitempty"`
	}

	poolMap map[string]pool

	refund struct {
		asset0  string
		asset1  string
		refunds [2]sdk.Int
	}
)

func ExportAstroportLockdrop(app *app.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	bl.RegisterAddress(AddressAstroportLockdrop, util.DenomLUNA)
	bl.RegisterAddress(AddressAstroportLockdrop, util.DenomUST)
	bl.RegisterAddress(AddressAstroportLockdrop, util.DenomAUST)

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
	poolsPrefix := util.GeneratePrefix("LiquidityPools")
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), lockdrop, poolsPrefix, func(key, value []byte) bool {
		var lpAddr = string(key)
		var pi poolInfo

		lpAddr = string(key)
		util.MustUnmarshalTMJSON(value, &pi)

		liquidityPools[lpAddr] = pi

		return false
	})

	// 3. figure out astroport pair addr for each terraswap lp token.
	// TerraswapLP => AstoportPair
	var pairAddresses = make(map[string]string)
	var minter struct {
		Minter string `json:"minter"`
	}
	for terraswapLPAddr, pi := range liquidityPools {
		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: pi.MigrationInfo.AstroportLPToken,
			QueryMsg:        []byte("{\"minter\": {}}"),
		}, &minter); err != nil {
			return nil, fmt.Errorf("failed to fetch minter: %v", err)
		}

		pairAddresses[terraswapLPAddr] = minter.Minter
	}

	// 4. Iterate over all lockdrop pos
	prefix := util.GeneratePrefix("lockup_position")
	var lockupInfo struct {
		LPUnitsLocked          sdk.Int `json:"lp_units_locked"`
		AstroportLPTransferred sdk.Int `json:"astroport_lp_transferred"`
	}
	var userRefunds = make(map[string][]refund)
	keeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), lockdrop, prefix, func(key, value []byte) bool {
		terraswapLPAddress := string(key[2:46])
		userAddress := string(key[48:92])

		util.MustUnmarshalTMJSON(value, &lockupInfo)

		pairPool := pm[pairAddresses[terraswapLPAddress]]
		refundAssets := getShareInAssets(pairPool, lockupInfo.LPUnitsLocked, pairPool.TotalShare)

		// create new slice if not initialized yet
		if userRefunds[userAddress] == nil {
			userRefunds[userAddress] = make([]refund, 0)
		}

		userRefunds[userAddress] = append(userRefunds[userAddress], refund{
			asset0:  pickDenomOrContractAddress(pairPool.Asset[0].AssetInfo),
			asset1:  pickDenomOrContractAddress(pairPool.Asset[1].AssetInfo),
			refunds: refundAssets,
		})

		return false
	})

	// 5. add up all pos and derive final balance
	var finalBalance = make(util.SnapshotBalanceAggregateMap)
	for userAddr, refunds := range userRefunds {
		userBalance := make([]util.SnapshotBalance, 0)

		for _, ref := range refunds {
			if asset0name, ok := coalesceToBalanceDenom(ref.asset0); ok {
				userBalance = append(userBalance, util.SnapshotBalance{
					Denom:   asset0name,
					Balance: ref.refunds[0],
				})
			}

			if asset1name, ok := coalesceToBalanceDenom(ref.asset1); ok {
				userBalance = append(userBalance, util.SnapshotBalance{
					Denom:   asset1name,
					Balance: ref.refunds[1],
				})
			}
		}

		finalBalance[userAddr] = userBalance
	}

	return finalBalance, nil
}

func getShareInAssets(p pool, lpAmount sdk.Int, totalShare sdk.Int) [2]sdk.Int {
	shareRatio := sdk.ZeroDec()
	if !totalShare.IsZero() {
		shareRatio = sdk.NewDecFromInt(lpAmount).Quo(sdk.NewDecFromInt(totalShare))
	}

	return [2]sdk.Int{
		shareRatio.MulInt(p.Asset[0].Amount).TruncateInt(),
		shareRatio.MulInt(p.Asset[1].Amount).TruncateInt(),
	}
}

// see if pool contains any of LUNA, UST, AUST, BLUNA
func isTargetPool(p *pool) bool {
	isOk := (p.Asset[0].AssetInfo.NativeToken != nil && p.Asset[0].AssetInfo.NativeToken.Denom == util.DenomLUNA) ||
		(p.Asset[0].AssetInfo.NativeToken != nil && p.Asset[0].AssetInfo.NativeToken.Denom == util.DenomUST) ||
		(p.Asset[0].AssetInfo.Token != nil && p.Asset[0].AssetInfo.Token.ContractAddr == AddressAUST) ||
		(p.Asset[0].AssetInfo.Token != nil && p.Asset[0].AssetInfo.Token.ContractAddr == AddressBLUNA) ||
		(p.Asset[1].AssetInfo.NativeToken != nil && p.Asset[1].AssetInfo.NativeToken.Denom == util.DenomLUNA) ||
		(p.Asset[1].AssetInfo.NativeToken != nil && p.Asset[1].AssetInfo.NativeToken.Denom == util.DenomUST) ||
		(p.Asset[1].AssetInfo.Token != nil && p.Asset[1].AssetInfo.Token.ContractAddr == AddressAUST) ||
		(p.Asset[1].AssetInfo.Token != nil && p.Asset[1].AssetInfo.Token.ContractAddr == AddressBLUNA)

	return isOk
}

func pickDenomOrContractAddress(asset assetInfo) string {
	if asset.Token != nil {
		return asset.Token.ContractAddr
	}

	if asset.NativeToken != nil {
		return asset.NativeToken.Denom
	}

	panic("unknown denom")
}

func coalesceToBalanceDenom(assetName string) (string, bool) {
	switch assetName {
	case "uusd":
		return util.DenomUST, true
	case "uluna":
		return util.DenomLUNA, true
	case AddressAUST:
		return util.DenomAUST, true
	case AddressBLUNA: // treat bLUNA the same as LUNA, deal with exchange rate later
		return util.DenomBLUNA, true
	}

	return "", false
}
