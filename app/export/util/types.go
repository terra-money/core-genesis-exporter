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

type Blacklist map[string][]string

func (bl Blacklist) RegisterAddress(denom string, address string) {
	bl[denom] = append(bl[denom], address)
}
