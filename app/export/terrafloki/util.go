package terrafloki

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app/export/util"
)

type (
	asset struct {
		AssetInfo assetInfo `json:"info"`
		Amount    sdk.Int   `json:"amount"`
	}

	pool struct {
		Assets     [2]asset `json:"assets"`
		TotalShare sdk.Int  `json:"total_share"`
	}

	pair struct {
		AssetInfos     [2]assetInfo `json:"asset_infos"`
		ContractAddr   []byte       `json:"contract_addr"`
		LiquidityToken string       `json:"liquidity_token"`
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
	pairMap map[string]pair

	refund struct {
		asset0  string
		asset1  string
		refunds [2]sdk.Int
	}

	tokenInfo struct {
		TotalSupply sdk.Int `json:"total_supply"`
	}
)

// see if pool contains any of LUNA, UST, AUST, BLUNA
func isTargetPool(p *pool) bool {
	isOk := (p.Assets[0].AssetInfo.NativeToken != nil && p.Assets[0].AssetInfo.NativeToken.Denom == util.DenomLUNA) ||
		(p.Assets[0].AssetInfo.NativeToken != nil && p.Assets[0].AssetInfo.NativeToken.Denom == util.DenomUST) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == util.AddressAUST) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == util.AddressBLUNA) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == util.AddressSTLUNA) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == util.AddressCLUNA) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == util.AddressPLUNA) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == util.AddressNLUNA) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == util.AddressSTEAK) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == util.AddressLUNAX) ||
		(p.Assets[1].AssetInfo.NativeToken != nil && p.Assets[1].AssetInfo.NativeToken.Denom == util.DenomLUNA) ||
		(p.Assets[1].AssetInfo.NativeToken != nil && p.Assets[1].AssetInfo.NativeToken.Denom == util.DenomUST) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == util.AddressAUST) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == util.AddressBLUNA) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == util.AddressSTLUNA) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == util.AddressCLUNA) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == util.AddressPLUNA) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == util.AddressNLUNA) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == util.AddressSTEAK) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == util.AddressLUNAX)

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
	case util.AddressAUST:
		return util.DenomAUST, true
	case util.AddressBLUNA:
		return util.DenomBLUNA, true
	case util.AddressSTLUNA:
		return util.DenomSTLUNA, true
	case util.AddressCLUNA:
		return util.DenomCLUNA, true
	case util.AddressPLUNA:
		return util.DenomPLUNA, true
	case util.AddressNLUNA:
		return util.DenomNLUNA, true
	case util.AddressSTEAK:
		return util.DenomSTEAK, true
	case util.AddressLUNAX:
		return util.DenomLUNAX, true
	}

	return "", false
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
