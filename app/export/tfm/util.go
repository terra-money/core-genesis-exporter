package tfm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app/export/util"
)

// see if pool contains any of LUNA, UST, AUST, BLUNA

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
