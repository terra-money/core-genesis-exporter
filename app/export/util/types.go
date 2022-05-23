package util

import sdk "github.com/cosmos/cosmos-sdk/types"

type BalanceMap map[string]sdk.Int
type SnapshotBalance struct {
	Denom   string  `json:"denom"`
	Balance sdk.Int `json:"balance"`
}
type SnapshotBalanceMap map[string]SnapshotBalance
