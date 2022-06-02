package astroport

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app/export/anchor"
)

var (
	AddressAstroportLockdrop = "terra1627ldjvxatt54ydd3ns6xaxtd68a2vtyu7kakj"
	AddressAstroportFactory  = "terra1fnywlw4edny3vw44x04xd67uzkdqluymgreu7g"
	PylonStaking             = "terra19nek85kaqrvzlxygw20jhy08h3ryjf5kg4ep3l"
	PylonLp                  = "terra16unvjel8vvtanxjpw49ehvga5qjlstn8c826qe"
	AstroUstLp               = "terra17n5sunn88hpy965mzvt3079fqx3rttnplg779g"
	AddressAstroportAuction  = "terra1tvld5k6pus2yh7pcu7xuwyjedn7mjxfkkkjjap"

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
		ContractAddr   string       `json:"contract_addr"`
		LiquidityToken string       `json:"liquidity_token"`
	}

	// only care about migration info to figure out ts -> astro migrated
	poolInfo struct {
		TerraswapAmountInLockup sdk.Int `json:"terraswap_amount_in_lockups"`
		MigrationInfo           struct {
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
	pairMap map[string]pair

	refund struct {
		asset0  string
		asset1  string
		refunds [2]sdk.Int
	}
)
