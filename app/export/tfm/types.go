package tfm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	AddressTfmFactory = "terra1u27ypputx3pu865luzs4fpjsj4llsnzf9qeq2p"

	StakingContracts = []string{
		"terra1nmheu06jz6mxsg5n8nnmyn6fxgnvxzk28femx2",
		"terra1vsd2sgk47tm262q7l3eq74u9rrek4ucn2w2jm5",
		"terra1ar2lwxegfnap7umqx9ljp0wytt3t76yyc8ys2z",
		"terra1uccez5grl2sr8xd4rswtgtc7ewf8ugfvcfutv5",
		"terra1x3fcujqgk8uvn8vssw52jtwtanlu3shgj3ftlk",
		"terra1rvvz9r8ys755wy9hat39es03d6h25rtz2u329l",
	}
	FarmContracts = []string{
		"terra14xg04lntty04vgqdcl8jnkclrkg8m532q6w2ga",
		"terra1fvt7pfnxqrc8fx45nc5weg2hjwfz3ayevt3gha",
		"terra13kq9rqxn0k252qfs4ww4zxzye83ypuye6w2hhm",
		"terra14czgy66f5wf9vxgvvmt5ajrv5urxvhm0359lrk",
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
	stakerInfo struct {
		Staker        string  `json:"staker"`
		BondAmount    sdk.Int `json:"bond_amount"`
		RewardIndex   sdk.Dec `json:"reward_index"`
		PendingReward sdk.Int `json:"pending_reward"`
	}
)
