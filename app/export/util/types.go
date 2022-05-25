package util

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (s SnapshotBalanceAggregateMap) AppendOrAddBalance(addr string, newBalance SnapshotBalance) {
	for i, balance := range s[addr] {
		if balance.Denom == newBalance.Denom {
			s[addr][i].Balance = s[addr][i].Balance.Add(newBalance.Balance)
			return
		}
	}
	s[addr] = append(s[addr], newBalance)
}

func (s SnapshotBalanceAggregateMap) ApplyBlackList(bl Blacklist) {
	for denom, addrList := range bl {
		for _, addr := range addrList {
			for i, snapshotBalance := range s[addr] {
				if snapshotBalance.Denom == denom && !snapshotBalance.Balance.IsZero() {
					fmt.Printf("Removed %s %s from %s\n", snapshotBalance.Balance, snapshotBalance.Denom, addr)
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
