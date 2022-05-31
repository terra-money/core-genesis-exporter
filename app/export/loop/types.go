package loop

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app/export/anchor"
)

var (
	AddressLoopFactory1 = "terra16hdjuvghcumu6prg22cdjl96ptuay6r0hc6yns"
	AddressLoopFactory2 = "terra10fp5e9m5avthm76z2ujgje2atw6nc87pwdwtww"
	//AddressLoopFarm1    = "terra1jqjpa66ethxc8wkkv5dvtvv7mp546expls6lw4"
	AddressLoopFarm1 = "terra1swgnlreprmfjxf2trul495uh4yphpkqucls8fv"
	AddressLoopFarm2 = "terra1cr7ytvgcrrkymkshl25klgeqxfs48dq4rv8j26"

	AddressAUST   = anchor.AddressAUST
	AddressBLUNA  = "terra1kc87mu460fwkqte29rquh4hc20m54fxwtsx7gp"
	AddressSTLUNA = "terra1yg3j2s986nyp5z7r2lvt0hx3r0lnd7kwvwwtsc"
	AddressSTEAK  = "terra1rl4zyexjphwgx6v3ytyljkkc4mrje2pyznaclv"
	AddressNLUNA  = "terra10f2mt82kjnkxqj2gepgwl637u2w4ue2z5nhz5j"
	AddressCLUNA  = "terra13zaagrrrxj47qjwczsczujlvnnntde7fdt0mau"
	AddressPLUNA  = "terra1tlgelulz9pdkhls6uglfn5lmxarx7f2gxtdzh2"
	AddressLUNAX  = "terra17y9qkl8dfkeg4py7n0g5407emqnemc3yqk5rup"
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
		LiquidityToken []byte       `json:"liquidity_token"`
	}

	assetInfo struct {
		Token *struct {
			ContractAddr string `json:"contract_addr"`
		} `json:"token"`
		NativeToken *struct {
			Denom string `json:"denom"`
		} `json:"native_token"`
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
