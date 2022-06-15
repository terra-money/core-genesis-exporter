package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Export struct {
	AppState struct {
		Swaps struct {
			PoolRecords  []Pool  `json:"pool_records"`
			ShareRecords []Share `json:"share_records"`
		} `json:"swap"`
		Cdp struct {
			Cdps []Cdp `json:"cdps"`
		} `json:"cdp"`
		Bank struct {
			Balances []Balances `json:"balances"`
		} `json:"bank"`
	} `json:"app_state"`
}

type Cdp struct {
	Owner      string  `json:"owner"`
	Collateral Balance `json:"collateral"`
}

type Pool struct {
	PoolId      string  `json:"pool_id"`
	ReservesA   Balance `json:"reserves_a"`
	ReservesB   Balance `json:"reserves_b"`
	TotalShares sdk.Int `json:"total_shares"`
}

type Share struct {
	Depositor   string  `json:"depositor"`
	PoolId      string  `json:"pool_id"`
	SharesOwned sdk.Int `json:"shares_owned"`
}

type Balance struct {
	Amount sdk.Int `json:"amount"`
	Denom  string  `json:"denom"`
}

type Balances struct {
	Address string    `json:"address"`
	Coins   []Balance `json:"coins"`
}
