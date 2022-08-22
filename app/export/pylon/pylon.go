package pylon

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	// stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/anchor"
	"github.com/terra-money/core/app/export/spectrum"
	util "github.com/terra-money/core/app/export/util"

	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	PylonPools = []string{
		// Mine
		"terra1z5j60wct88yz62ylqa4t8p8239cwx9kjlghkg2",
		// Loop
		"terra149fxy4crxnhy4z2lezefwe7evjthlsttyse20m",
		// TWD
		"terra1he8j44cv2fcntujjnsqn3ummauua555agz4js0",
		// VKR
		"terra1zxtcxxjqp7c46g8jx0t25s5ysa5qawmwd2w7nr",
		// GP
		"terra1jk0xh49ft2ls4u9dlfqweed8080u6ysumvmtcz",
		// ORION
		"terra15y9r79wlu8uqvlu3k7vgv0kgdy29m8j9tt9xgg",
		// Deviants Faction
		"terra19zn5u7ej083em99was4y02j3yhracnxwxcvmt4",
		// Whale
		"terra15y70slq4l4s5da2etsyqasyjht0dnquj03qm05",
		// Glow
		"terra1g9kzlt58ycppx9elymnrgxmwssfawym668r2y4",
		// SAYVE
		"terra1he8ymhmqmtuu5699akwk94urap6z09xnnews32",
		// XDEFI
		"terra1vftcl08p73v3nkuwvv5ntznku44s7p2tq00mgn",
		// Luna Bulls
		"terra132u62nsympysvtg3nng5xg6tjf6cr8sxrq7ena",
		// TerraBots
		"terra1dyattlzq58ty7pat337a9dz6j46thldu5gn8ls",
		// Lunart
		"terra1xkw8vusucy9c2w9hxuw6lktxk2w8g72utdyq96",
	}

	PSIAnchorPool = "terra1xu84jh7x2ugt3gkpv8d450hdwcyejtcwwkkzgv"
	PSIPool       = "terra1fmnedmd3732gwyyj47r5p03055mygce98dpte2"
	PsiSpecFarm   = "terra1kr82wxlvg773vjay95epyckna9g4vppjyfxgd0"
	BPsiDpLp      = "terra1t4uuc09t4ld560vg2k9w2f5m9e5trftnym50zj"
	BPsiDpPair    = "terra167gwjhv4mrs0fqj0q5tejyl6cz6qc2cl95z530"
	BPsiDpToken   = "terra1zsaswh926ey8qa5x4vj93kzzlfnef0pstuca0y"
	PsiDpToken    = "terra1rzj8fua8wmqq7x0ka8emr6t7n9j45u82pe6sgc"

	// Contracts found on https://api.pylon.money/api/gateway/v1/projects/
	PylonLookup = map[string][]string{
		"terra1z5j60wct88yz62ylqa4t8p8239cwx9kjlghkg2": {
			"terra19vnwdqz4um0z8f69pc8y0z4ncrcxm4cjf3gevz",
			"terra1t3wtg074jjscqc5k2hn6l4lsremccm25tt77zp",
			"terra1za627n8zc8wqg06n9h7khpmjcnlkdkt38rkl3u",
		},
		"terra149fxy4crxnhy4z2lezefwe7evjthlsttyse20m": {
			"terra1p9ns8r3unhvvp6ka3r52h79d0t6wjthxu4dfrs",
			"terra1527ks2mulus98lwx7qhdmrv4hug9pgx0m3c95s",
			"terra1jtd00mrwdpa6aecvw60mhzrqup054q054u53ch",
		},
		"terra1he8j44cv2fcntujjnsqn3ummauua555agz4js0": {
			"terra1qz6kp8nu5cqy6g679epd2f436p8uyry0aevrxc",
			"terra14v4g46j8ah9lpwwrnhxh6kyqvytmwd3ma9qvtu",
			"terra1fyduwdy0ncz8qur0rzp2t7skt4sc3e20w0d7qx",
		},
		"terra1xu84jh7x2ugt3gkpv8d450hdwcyejtcwwkkzgv": {
			"terra1fmnedmd3732gwyyj47r5p03055mygce98dpte2",
		},
		"terra1zxtcxxjqp7c46g8jx0t25s5ysa5qawmwd2w7nr": {
			"terra1p625agkeu4vrr4fqnl5c82myhy3z95t6tqycku",
			"terra1ftl4pt3l3ccjgk4ucndsff73uum888a2kcy779",
			"terra1yznc2p9q2smx8ku8m20hhv8amdmcj0zcvjh6km",
		},
		"terra1jk0xh49ft2ls4u9dlfqweed8080u6ysumvmtcz": {
			"terra10jrv8wy6s06mku9t6yawt2yr09wjlqsw0qk0vf",
		},
		"terra15y9r79wlu8uqvlu3k7vgv0kgdy29m8j9tt9xgg": {
			"terra1kmvvzp0vadlr4fvug6zve7736ufnt2x7rvdufm",
		},
		"terra19zn5u7ej083em99was4y02j3yhracnxwxcvmt4": {
			"terra1nata7lxk6ylx7ttu56jgp2s57g3fucl8saz0qw",
		},
		"terra15y70slq4l4s5da2etsyqasyjht0dnquj03qm05": {
			"terra1srw8lgcp4uqyar22ldeje5p0nx35q2jd93dc3k",
			"terra1ezxapxsduvp7v3njpcvtadwgwne3ch0muhce6u",
			"terra1wzer7q9zsug8jxgwp7l6dzd7ehc37nwg9fadef",
		},
		"terra1g9kzlt58ycppx9elymnrgxmwssfawym668r2y4": {
			"terra1nu4nxjjgw553zhc0k624h7vqmk5z5tj8ufrrzd",
			"terra1vwtr0trqz4nuqwy2g2n3szwczp2a4ccsf8hn9j",
			"terra1709w9ll57sdmyr8zzqtp423r6cwxyc33hc9xnq",
		},
		"terra1he8ymhmqmtuu5699akwk94urap6z09xnnews32": {
			"terra14hg497r875c62f4kxs8q7pkek0kdw0dphppa0h",
			"terra1xgewsvsl2gff63fplma52smt8r76fyzvurfwcm",
			"terra1s9qs7r8aacs0auynhumdn4jtgju58kjlvrg6uw",
		},
		"terra1vftcl08p73v3nkuwvv5ntznku44s7p2tq00mgn": {
			"terra1a9cu63vx3u0m386x5f74qsr7sw405zdj5uhpll",
			"terra14e0g6gqldl2ruyt6ps72gwl2xc6lvxh7mz02lw",
			"terra1r9we2p8knhzxn0gk0ak667fcxdv26x4tp46l4k",
		},
		"terra132u62nsympysvtg3nng5xg6tjf6cr8sxrq7ena": {
			"terra1qt4wtj528s35kdk5zdylmwvt4nh7te5ets033t",
		},
		"terra1dyattlzq58ty7pat337a9dz6j46thldu5gn8ls": {
			"terra1cl8r8srtkj6k65kc8jalfnx5u5eewyq5wg2u5u",
		},
		"terra1xkw8vusucy9c2w9hxuw6lktxk2w8g72utdyq96": {
			"terra1ssq90m5juvxwukdxjjsd5un0sql7858wpa6h87",
			"terra19ky7jkpzkdner969tmpd9ury6y59sp0qs6pu9e",
			"terra1l3ajy0cq5vskww5jee7cg33qu455j8lg8cy9wv",
		},
		"terra1jzsjs8qx9ehsukzea9smuqtfuklmngmeh5csl3": {
			"terra1up7f322g6lcr0f3lak3dk23tpea2djercv0zpf",
			"terra158q2xv2uecal2nl8az3a33xrn4n2wr4m78zwe3",
			"terra162k36uafvezjyy5wawq7kmeqyr9e2juemkj9lc",
		},
	}
)

type PylonPoolConfig struct {
	Pool       string
	AUstAmount sdk.Int
	UstAmount  sdk.Int
	PoolToken  string `json:"dp_token"`
}

func ExportContract(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Pylon")
	var _ wasmtypes.QueryServer
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	snapshot := make(util.SnapshotBalanceAggregateMap)

	for _, pool := range PylonPools {
		config, err := getConfig(ctx, q, app.BankKeeper, pool)
		if err != nil {
			return nil, err
		}

		tokenBalances := make(map[string]sdk.Int)
		err = util.GetCW20AccountsAndBalances2(ctx, app.WasmKeeper, config.PoolToken, tokenBalances)
		if err != nil {
			return nil, err
		}
		aUstER, err := anchor.GetAUstExchangeRate(app)
		if err != nil {
			return nil, err
		}

		for address := range tokenBalances {
			for _, individualPool := range PylonLookup[pool] {
				var stakedBalance struct {
					Amount sdk.Int `json:"amount"`
				}
				err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
					ContractAddress: individualPool,
					QueryMsg:        []byte(fmt.Sprintf("{\"balance_of\":{\"owner\":\"%s\"}}", address)),
				}, &stakedBalance)
				if err != nil {
					return snapshot, err
				}
				if !stakedBalance.Amount.IsZero() {
					snapshot.AppendOrAddBalance(address, util.SnapshotBalance{
						Denom:   util.DenomAUST,
						Balance: stakedBalance.Amount.ToDec().Quo(aUstER).TruncateInt(),
					})
				}
			}
		}
	}
	return snapshot, nil
}

func ExportPsiPool(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	q := util.PrepWasmQueryServer(app)
	ctx := util.PrepCtx(app)
	prefix := util.GeneratePrefix("reward")
	// userLpHoldings := make(map[string]lpHoldings)
	walletSeen := make(map[string]bool)
	farmAddr, err := sdk.AccAddressFromBech32(PsiSpecFarm)
	if err != nil {
		return nil, err
	}
	aUstER, err := anchor.GetAUstExchangeRate(app)
	if err != nil {
		return nil, err
	}
	snapshot := make(util.SnapshotBalanceAggregateMap)
	app.WasmKeeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), farmAddr, prefix, func(key, value []byte) bool {
		walletAddress := sdk.AccAddress(key[2:22])
		if walletSeen[walletAddress.String()] {
			return false
		}
		walletSeen[walletAddress.String()] = true
		var rewards spectrum.RewardInfo
		err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: farmAddr.String(),
			QueryMsg:        []byte(fmt.Sprintf("{\"reward_info\":{\"staker_addr\":\"%s\"}}", walletAddress)),
		}, &rewards)
		if err != nil {
			panic(err)
		}
		for _, reward := range rewards.RewardInfo {
			amount := reward.LpAmount.ToDec().Quo(aUstER).TruncateInt()
			if !amount.IsZero() {
				snapshot.AppendOrAddBalance(walletAddress.String(), util.SnapshotBalance{
					Denom:   util.DenomAUST,
					Balance: amount,
				})
			}
		}
		return false
	})

	var pool struct {
		Assets []struct {
			TokenInfo struct {
				Token struct {
					ContractAddr string `json:"contract_addr"`
				} `json:"token"`
			} `json:"info"`
			Amount sdk.Int `json:"amount"`
		} `json:"assets"`
		TotalShare sdk.Int `json:"total_share"`
	}

	err = util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: BPsiDpPair,
		QueryMsg:        []byte("{\"pool\":{}}"),
	}, &pool)
	if err != nil {
		return nil, err
	}
	if pool.Assets[0].TokenInfo.Token.ContractAddr != BPsiDpToken {
		return nil, fmt.Errorf("unknown contract")
	}
	bPsiDpTotal := pool.Assets[0].Amount.ToDec()
	lpBalances := make(map[string]sdk.Int)
	lpTotal := pool.TotalShare.ToDec()
	err = util.GetCW20AccountsAndBalances2(ctx, app.WasmKeeper, BPsiDpLp, lpBalances)
	if err != nil {
		return nil, err
	}

	for addr, balance := range lpBalances {
		austBalance := balance.ToDec().Quo(lpTotal).Mul(bPsiDpTotal).Quo(aUstER).TruncateInt()
		if !austBalance.IsZero() {
			snapshot.AppendOrAddBalance(addr, util.SnapshotBalance{
				Denom:   util.DenomAUST,
				Balance: austBalance,
			})
		}
	}

	tokenHolders := []string{}
	limit := 10
	startAfter := ""
	for {
		var query string
		if startAfter == "" {
			query = fmt.Sprintf("{\"all_accounts\":{\"limit\": %d}}", limit)
		} else {
			query = fmt.Sprintf("{\"all_accounts\":{\"limit\": %d, \"start_after\": \"%s\"}}", limit, startAfter)
		}
		var allAccountResponse struct {
			Accounts []string `json:"accounts"`
		}
		err = util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: BPsiDpToken,
			QueryMsg:        []byte(query),
		}, &allAccountResponse)
		if err != nil {
			return nil, err
		}
		tokenHolders = append(tokenHolders, allAccountResponse.Accounts...)
		if limit > len(allAccountResponse.Accounts) {
			break
		}
		startAfter = allAccountResponse.Accounts[len(allAccountResponse.Accounts)-1]
	}

	psiDpTokens := make(map[string]sdk.Int)
	err = util.GetCW20AccountsAndBalances2(ctx, app.WasmKeeper, PsiDpToken, psiDpTokens)
	if err != nil {
		return nil, err
	}

	for _, addr := range tokenHolders {
		if psiDpTokens[addr].IsNil() {
			// Set any amount here just to make sure we account for them later.
			// Amount is not used for aUST calculation
			psiDpTokens[addr] = sdk.NewInt(0)
		}
	}

	for address := range psiDpTokens {
		var stakedBalance struct {
			Amount sdk.Int `json:"amount"`
		}
		err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: PSIPool,
			QueryMsg:        []byte(fmt.Sprintf("{\"balance_of\":{\"owner\":\"%s\"}}", address)),
		}, &stakedBalance)
		if err != nil {
			return snapshot, err
		}
		if !stakedBalance.Amount.IsZero() {
			snapshot.AppendOrAddBalance(address, util.SnapshotBalance{
				Denom:   util.DenomAUST,
				Balance: stakedBalance.Amount.ToDec().Quo(aUstER).TruncateInt(),
			})
		}
	}

	cbl := util.Blacklist{}
	cbl.RegisterAddress(util.DenomAUST, BPsiDpPair)
	cbl.RegisterAddress(util.DenomAUST, PsiSpecFarm)
	snapshot.ApplyBlackList(cbl)

	var austInPool struct {
		Balance sdk.Int `json:"balance"`
	}

	util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: util.AddressAUST,
		QueryMsg:        []byte(fmt.Sprintf("{\"balance\": {\"address\": \"%s\"}}", PSIAnchorPool)),
	}, &austInPool)

	var rewards struct {
		Amount sdk.Int `json:"amount"`
		Fee    sdk.Int `json:"fee"`
	}
	err = util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: PSIAnchorPool,
		QueryMsg:        []byte("{\"claimable_reward\":{}}"),
	}, &rewards)
	if err != nil {
		return nil, err
	}

	// err = util.AlmostEqual("aUST in bPSIDp24m pool doesnt match", snapshot.SumOfDenom(util.DenomAUST), austInPool.Balance.Sub(rewards.Amount), sdk.NewInt(10000000))
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func Audit(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	app.Logger().Info("Audit -- Pylon")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	totalAUst := sdk.NewInt(0)

	aUstER, err := anchor.GetAUstExchangeRate(app)
	if err != nil {
		return err
	}

	for _, pool := range PylonPools {
		config, err := getConfig(ctx, q, app.BankKeeper, pool)
		if err != nil {
			return err
		}

		totalAUst = totalAUst.Add(config.UstAmount.ToDec().QuoTruncate(aUstER).TruncateInt()).Add(config.AUstAmount)
	}

	if err := util.AlmostEqual(util.DenomUST, totalAUst, snapshot.SumOfDenom(util.DenomAUST), sdk.NewInt(100000000000)); err != nil {
		return err
	}

	return nil
}

func getConfig(ctx context.Context, q wasmtypes.QueryServer, k wasmtypes.BankKeeper, pool string) (PylonPoolConfig, error) {
	var config PylonPoolConfig
	err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: pool,
		QueryMsg:        []byte("{\"config\":{}}"),
	}, &config)
	config.Pool = pool
	if err != nil {
		return config, err
	}
	aUstBalance, err := util.GetCW20Balance(ctx, q, util.AUST, pool)
	if err != nil {
		return config, err
	}
	config.AUstAmount = aUstBalance
	ustBalance, err := util.GetNativeBalance(ctx, k, util.DenomUST, pool)
	if err != nil {
		return config, err
	}
	config.UstAmount = ustBalance
	return config, err
}
