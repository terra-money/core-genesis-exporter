package main

import (
	"encoding/json"
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-money/core/scripts/kava/types"
)

var (
	interestedPools = map[string]bool{
		"ibc/B448C0CA358B958301D328CCDC5D5AD642FC30A6D3AE106FF721DB315F3DDE5C:usdx": true, // ust - usdx
		"ibc/B8AF5D92165F35AB31F3FC7C7B444B9D240760FA5D406C49D24862BD0284E395:usdx": true, // luna -usdx
	}
	interestedDenoms = map[string]bool{
		"ibc/B448C0CA358B958301D328CCDC5D5AD642FC30A6D3AE106FF721DB315F3DDE5C": true, // UST
		"ibc/B8AF5D92165F35AB31F3FC7C7B444B9D240760FA5D406C49D24862BD0284E395": true, // LUNA
	}

	ibcToDenom = map[string]string{
		"ibc/B448C0CA358B958301D328CCDC5D5AD642FC30A6D3AE106FF721DB315F3DDE5C": "uusd",
		"ibc/B8AF5D92165F35AB31F3FC7C7B444B9D240760FA5D406C49D24862BD0284E395": "uluna",
	}
)

func main() {
	var kavaState types.Export
	pathToJson := os.Args[1]
	if pathToJson == "" {
		panic("json file not specified")
	}
	kavaStateRaw, err := os.ReadFile(pathToJson)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(kavaStateRaw, &kavaState)
	if err != nil {
		panic(err)
	}

	swapHolders, err := GetSwapHolders(kavaState.AppState.Swaps.PoolRecords, kavaState.AppState.Swaps.ShareRecords)
	if err != nil {
		panic(err)
	}

	cdpHolders, err := GetCdpHolders(kavaState.AppState.Cdp.Cdps)
	if err != nil {
		panic(err)
	}

	balanceHolders, err := GetHolders(kavaState.AppState.Bank.Balances)
	if err != nil {
		panic(err)
	}
	holders := mergeHolders(swapHolders, cdpHolders, balanceHolders)
	delete(holders, "kava1mfru9azs5nua2wxcd4sq64g5nt7nn4n8s2w8cu")
	fmt.Println("address,denom,amount")
	for addr, holding := range holders {
		for _, b := range holding {
			fmt.Printf("%s,%s,%s\n", addr, ibcToDenom[b.Denom], b.Amount)
		}
	}
}

func GetSwapHolders(pools []types.Pool, shares []types.Share) (map[string][]types.Balance, error) {
	poolMap := make(map[string]types.Pool)
	for _, p := range pools {
		poolMap[p.PoolId] = p
	}

	sharesMap := make(map[string]map[string]sdk.Int)
	for _, s := range shares {
		if interestedPools[s.PoolId] {
			if sharesMap[s.PoolId] == nil {
				sharesMap[s.PoolId] = make(map[string]sdk.Int)
			}
			sharesMap[s.PoolId][s.Depositor] = s.SharesOwned
		}
	}

	holders := make(map[string][]types.Balance)
	for poolId, depositors := range sharesMap {
		var reserve sdk.Int
		var denom string
		pool := poolMap[poolId]
		if interestedDenoms[pool.ReservesA.Denom] {
			denom = pool.ReservesA.Denom
			reserve = pool.ReservesA.Amount
		}
		if interestedDenoms[pool.ReservesB.Denom] {
			denom = pool.ReservesB.Denom
			reserve = pool.ReservesB.Amount
		}

		if reserve.IsNil() {
			return nil, fmt.Errorf("unknown reserve %s, a: %s, b: %s", poolId, pool.ReservesA.Denom, pool.ReservesB.Denom)
		}
		for addr, share := range depositors {
			holders[addr] = append(holders[addr], types.Balance{
				Denom:  denom,
				Amount: share.Mul(reserve).Quo(pool.TotalShares),
			})
		}
	}

	return holders, nil
}

func GetCdpHolders(cdps []types.Cdp) (map[string][]types.Balance, error) {
	holders := make(map[string][]types.Balance)
	for _, c := range cdps {
		if interestedDenoms[c.Collateral.Denom] {
			holders[c.Owner] = append(holders[c.Owner], types.Balance{
				Denom:  c.Collateral.Denom,
				Amount: c.Collateral.Amount,
			})
		}
	}
	return holders, nil
}

func GetHolders(balances []types.Balances) (map[string][]types.Balance, error) {
	holders := make(map[string][]types.Balance)
	for _, b := range balances {
		for _, c := range b.Coins {
			if interestedDenoms[c.Denom] {
				holders[b.Address] = append(holders[b.Address], types.Balance{
					Denom:  c.Denom,
					Amount: c.Amount,
				})
			}
		}
	}
	return holders, nil
}

func mergeHolders(bbs ...map[string][]types.Balance) map[string][]types.Balance {
	holders := make(map[string][]types.Balance)
	for _, bb := range bbs {
		for addr, b := range bb {
			for _, c := range b {
				found := false
				for j, h := range holders[addr] {
					if h.Denom == c.Denom {
						holders[addr][j] = types.Balance{
							Amount: c.Amount.Add(h.Amount),
							Denom:  c.Denom,
						}
						found = true
					}
				}
				if !found {
					holders[addr] = append(holders[addr], types.Balance{
						Amount: c.Amount,
						Denom:  c.Denom,
					})
				}
			}
		}
	}
	return holders
}
