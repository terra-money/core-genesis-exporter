package util

import sdk "github.com/cosmos/cosmos-sdk/types"

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
