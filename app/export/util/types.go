package util

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

type BalanceMap map[string]sdk.Int
type SnapshotBalance struct {
	Denom   string  `json:"denom"`
	Balance sdk.Int `json:"balance"`
}

func (b *SnapshotBalance) AddInto(i sdk.Int) {
	b.Balance = b.Balance.Add(i)
}

type SnapshotBalanceMap map[string]SnapshotBalance
type SnapshotBalanceAggregateMap map[string][]SnapshotBalance

// map[denom][]address
type Blacklist map[string][]string

func (bl Blacklist) RegisterAddress(denom string, address string) {
	bl[denom] = append(bl[denom], address)
}

func (bl Blacklist) GetAddressesByDenom(denom string) []string {
	return bl[denom]
}

func MergeSnapshots(ss ...SnapshotBalanceAggregateMap) (s3 SnapshotBalanceAggregateMap) {
	s3 = make(SnapshotBalanceAggregateMap)
	for _, s := range ss {
		for addr, sbs := range s {
			for _, sbs := range sbs {
				s3.AppendOrAddBalance(addr, sbs)
			}
		}
	}
	return s3
}

func (s SnapshotBalanceAggregateMap) AppendOrAddBalance(addr string, newBalance SnapshotBalance) {
	for i, balance := range s[addr] {
		if balance.Denom == newBalance.Denom {
			s[addr][i].Balance = s[addr][i].Balance.Add(newBalance.Balance)
			return
		}
	}
	s[addr] = append(s[addr], newBalance)
}

func (s SnapshotBalanceAggregateMap) GetAddrBalance(addr string, denom string) sdk.Int {
	sum := sdk.NewInt(0)
	for _, balance := range s[addr] {
		if balance.Denom == denom {
			sum = sum.Add(balance.Balance)
		}
	}
	return sum
}

func (bl Blacklist) GetAddressesByDenomMap(denom string) map[string]bool {
	list := bl[denom]

	m := make(map[string]bool)
	for _, addr := range list {
		m[addr] = true
	}

	return m
}

func (s SnapshotBalanceAggregateMap) SumOfDenom(denom string) sdk.Int {
	sum := sdk.NewInt(0)
	for _, balances := range s {
		for _, balance := range balances {
			if !balance.Balance.IsNil() && balance.Denom == denom {
				sum = sum.Add(balance.Balance)
			}
		}
	}
	return sum
}

func (s SnapshotBalanceAggregateMap) Add(balances map[string]sdk.Int, denom string) {
	for a, b := range balances {
		s[a] = append(s[a], SnapshotBalance{
			Denom:   denom,
			Balance: b,
		})
	}
}

func (s SnapshotBalanceAggregateMap) FilterByDenom(denom string) map[string]sdk.Int {
	filtered := make(map[string]sdk.Int)
	for w, sbs := range s {
		for _, sb := range sbs {
			if sb.Denom == denom && sb.Balance.IsPositive() {
				if filtered[w].IsNil() {
					filtered[w] = sb.Balance
				} else {
					filtered[w] = filtered[w].Add(sb.Balance)
				}
			}
		}
	}
	return filtered
}

func (s SnapshotBalanceAggregateMap) PickDenomIntoBalanceMap(denom string) BalanceMap {
	v := make(BalanceMap)
	for addr, holdings := range s {

		for _, holding := range holdings {
			if holding.Denom == denom {
				v[addr] = holding.Balance
			}
		}
	}

	return v
}

func (s SnapshotBalanceAggregateMap) ApplyBlackList(bl Blacklist) {
	for denom, addrList := range bl {
		for _, addr := range addrList {
			for i, snapshotBalance := range s[addr] {
				if snapshotBalance.Denom == denom && !snapshotBalance.Balance.IsZero() {
					// fmt.Printf("Removed %s %s from %s\n", snapshotBalance.Balance, snapshotBalance.Denom, addr)
					// Remove by setting to 0
					s[addr][i] = SnapshotBalance{
						Denom:   snapshotBalance.Denom,
						Balance: sdk.NewInt(0),
					}
				}
			}
		}
	}
}

func (s SnapshotBalanceAggregateMap) ExportToBalances() []types.Balance {
	// merge just to make sure all balances are collapsed
	merged := MergeSnapshots(s)

	var export []types.Balance
	for addr, balances := range merged {
		account := types.Balance{
			Address: addr,
		}
		for _, balance := range balances {
			account.Coins = append(account.Coins, sdk.Coin{
				Amount: balance.Balance,
				Denom:  balance.Denom,
			})
		}
	}
	return export
}
