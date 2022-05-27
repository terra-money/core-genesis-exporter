package util

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMergeSnapshot(t *testing.T) {
	s1 := SnapshotBalanceAggregateMap{
		"addr1": []SnapshotBalance{
			{
				Denom:   DenomAUST,
				Balance: sdk.NewInt(100),
			},
			{
				Denom:   DenomAUST,
				Balance: sdk.NewInt(100),
			},
		},
	}
	s2 := SnapshotBalanceAggregateMap{
		"addr1": []SnapshotBalance{
			{
				Denom:   DenomAUST,
				Balance: sdk.NewInt(200),
			},
		},
	}
	s3 := SnapshotBalanceAggregateMap{
		"addr1": []SnapshotBalance{
			{
				Denom:   DenomAUST,
				Balance: sdk.NewInt(100),
			},
		},
	}

	s4 := MergeSnapshots(s1, s2, s3)

	if len(s4["addr1"]) > 1 {
		t.Fail()
	}

	if !s4["addr1"][0].Balance.Equal(sdk.NewInt(500)) {
		t.Fail()
	}
}

func TestMergeSnapshotMultipleBalances(t *testing.T) {
	s1 := SnapshotBalanceAggregateMap{
		"addr1": []SnapshotBalance{
			{
				Denom:   DenomAUST,
				Balance: sdk.NewInt(100),
			},
			{
				Denom:   DenomUST,
				Balance: sdk.NewInt(100),
			},
		},
	}
	s2 := SnapshotBalanceAggregateMap{
		"addr1": []SnapshotBalance{
			{
				Denom:   DenomAUST,
				Balance: sdk.NewInt(200),
			},
		},
	}
	s3 := SnapshotBalanceAggregateMap{
		"addr1": []SnapshotBalance{
			{
				Denom:   DenomAUST,
				Balance: sdk.NewInt(100),
			},
			{
				Denom:   DenomCLUNA,
				Balance: sdk.NewInt(200),
			},
		},
	}

	s4 := MergeSnapshots(s1, s2, s3)

	if len(s4["addr1"]) > 3 {
		t.Fail()
	}

	if !s4["addr1"][0].Balance.Equal(sdk.NewInt(400)) {
		t.Fail()
	}
	if !s4["addr1"][1].Balance.Equal(sdk.NewInt(100)) {
		t.Fail()
	}
	if !s4["addr1"][2].Balance.Equal(sdk.NewInt(200)) {
		t.Fail()
	}
}
