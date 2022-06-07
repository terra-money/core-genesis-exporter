package astroport

import "github.com/terra-money/core/app/export/util"

var (
	StakingContracts = []string{
		"terra1fmu29xhg5nk8jr0p603y5qugpk2r0ywcyxyv7k",
		"terra13n2sqaj25ugkt79k3evhvua30ut9qt8q0268zc",
		"terra1sxzggeujnxrd7hsx7uf2l6axh2uuv4zz5jadyg",
		"terra1h3mf22jm68ddueryuv2yxwfmqxxadvjceuaqz6",
		"terra1cw00274wlje5z8vtlrpaqx5cwj29c5a5ku2zhv",
		"terra12vu0rxec60rwg82hlkwdjnqwxrladt00rpllzl",
		"terra1wjc6zd6ue5sqmyucdu8erxj5cdf783tqle6dja",
		"terra15p807wnm9q3dyw4rvfqsaukxqt6lkuqe62q3mp",
		"terra188xjhn8h39ert7ezs0m2dlgsqd4vf6k6hmv4jw",
		"terra1za0ltkcxjpvfw8wnwhetj5mr5r05pl6dgy936g",
		"terra1r2ucpn7j8qcgvsvkzxr3x0698megrn2kn9nfwq",
		"terra1z5uvpz8ny5tz2lng30ff0aqnm5uuvxaat6lwxm",
		"terra10t8rn7swtkmkfm56mmxwmk2v9xrv78fljsd3ez",
		"terra19nek85kaqrvzlxygw20jhy08h3ryjf5kg4ep3l",
		"terra1gmggdadphqxua2kewcgn2l57xxteafpne50je0",
		"terra100yeqvww74h4yaejj6h733thgcafdaukjtw397",
		"terra17f7zu97865jmknk7p2glqvxzhduk78772ezac5",
		"terra1tyjfrx40kgpmf6mq2kyv6njgg59fxpv7pk8dhd",
		"terra1g7jjjkt5uvkjeyhp8ecdz4e4hvtn83sud3tmh2",
		"terra1x7v7qvumfl36g5jh0mtqx3c4g8c35sn0sqfuqp",
	}
)

// see if pool contains any of LUNA, UST, AUST, BLUNA
func isTargetPool(p *pool) bool {
	isOk := (p.Assets[0].AssetInfo.NativeToken != nil && p.Assets[0].AssetInfo.NativeToken.Denom == util.DenomLUNA) ||
		(p.Assets[0].AssetInfo.NativeToken != nil && p.Assets[0].AssetInfo.NativeToken.Denom == util.DenomUST) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == AddressAUST) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == AddressBLUNA) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == AddressSTLUNA) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == AddressCLUNA) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == AddressPLUNA) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == AddressNLUNA) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == AddressSTEAK) ||
		(p.Assets[0].AssetInfo.Token != nil && p.Assets[0].AssetInfo.Token.ContractAddr == AddressLUNAX) ||
		(p.Assets[1].AssetInfo.NativeToken != nil && p.Assets[1].AssetInfo.NativeToken.Denom == util.DenomLUNA) ||
		(p.Assets[1].AssetInfo.NativeToken != nil && p.Assets[1].AssetInfo.NativeToken.Denom == util.DenomUST) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == AddressAUST) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == AddressBLUNA) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == AddressSTLUNA) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == AddressCLUNA) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == AddressPLUNA) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == AddressNLUNA) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == AddressSTEAK) ||
		(p.Assets[1].AssetInfo.Token != nil && p.Assets[1].AssetInfo.Token.ContractAddr == AddressLUNAX)

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
	case AddressBLUNA:
		return util.DenomBLUNA, true
	case AddressSTLUNA:
		return util.DenomSTLUNA, true
	case AddressCLUNA:
		return util.DenomCLUNA, true
	case AddressPLUNA:
		return util.DenomPLUNA, true
	case AddressNLUNA:
		return util.DenomNLUNA, true
	case AddressSTEAK:
		return util.DenomSTEAK, true
	case AddressLUNAX:
		return util.DenomLUNAX, true
	}

	return "", false
}
