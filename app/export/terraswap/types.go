package terraswap

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/app/export/anchor"
)

var (
	AddressTerraswapFactory = "terra1ulgw0td86nvs4wtpsc80thv6xelk76ut7a7apj"

	AddressAUST   = anchor.AddressAUST
	AddressBLUNA  = "terra1kc87mu460fwkqte29rquh4hc20m54fxwtsx7gp"
	AddressSTLUNA = "terra1yg3j2s986nyp5z7r2lvt0hx3r0lnd7kwvwwtsc"
	AddressSTEAK  = "terra1rl4zyexjphwgx6v3ytyljkkc4mrje2pyznaclv"
	AddressNLUNA  = "terra10f2mt82kjnkxqj2gepgwl637u2w4ue2z5nhz5j"
	AddressCLUNA  = "terra13zaagrrrxj47qjwczsczujlvnnntde7fdt0mau"
	AddressPLUNA  = "terra1tlgelulz9pdkhls6uglfn5lmxarx7f2gxtdzh2"
	AddressLUNAX  = "terra17y9qkl8dfkeg4py7n0g5407emqnemc3yqk5rup"

	StakingContracts = []string{
		"terra1euaquddnk5eq495x7jjv0c8d5aldx39jeffsxh",
		"terra1a7fwra93sw8xy5wz779crks07u3ttf3u4mslfp",
		"terra160jxfrcwrn6nn5ns3zc9qj9c3rnkzlhnnghr9x",
		"terra1cf9q9lq7tdfju95sdw78y9e34a6qrq3rrc6dre",
		"terra175ueft8w9vpkj9jvehyjhkpk6szy8unq4lns8n",
		"terra1hs4ev0ghwn4wr888jwm56eztfpau6rjcd8mczc",
		"terra1ude6ggsvwrhefw2dqjh4j6r7fdmu9nk6nf2z32",
		"terra12kzewegufqprmzl20nhsuwjjq6xu8t8ppzt30a",
		"terra1jprduh6mc5tamy08pvw4c7wh7m7gcu2ej0c407",
		"terra1snra29afr9efzt6l34wfnhj3jn90hq6rx8jhje",
		"terra1gw80vdfk028d5qywl86c4culg7q286748c0thz",
		"terra1pafra8qj9efnxhv5jq7n4qy0mq5pge8w4k8s2k",
		"terra1pyl3u0v0y7szlj8njctkhys9fvtsl6wva00fd5",
		"terra10eyxljyqkcvhs4dgr534hk0wehc28tz6gwnh8a",
		"terra1897an2xux840p9lrh6py3ryankc6mspw49xse3",
		"terra1munkej78p9ckry0p6vvkkk94c89tnyjhnxfskl",
		"terra1le3a67j4khkjhyytkllxre60dvywm43ztq2s8t",
		"terra1u94zwrreyz3t0jx25nl7800pxsrk6e6dwjqpsx",
		"terra19nek85kaqrvzlxygw20jhy08h3ryjf5kg4ep3l",
		"terra1n9a0y0gyn87wnwwa7p8rpjhkljcsls6kvaywc8",
		"terra1fvq04l6pzslhxf7yyxhfmz5hv64xsye8vcmgzt",
		"terra1hxyyjpu8548ccwth9pnc5ztgpupnn2a3c9s0f8",
	}
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
