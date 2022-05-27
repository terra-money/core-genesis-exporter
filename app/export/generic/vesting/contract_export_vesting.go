package vesting

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/generic/common"
	"github.com/terra-money/core/app/export/util"
	wasmkeeper "github.com/terra-money/core/x/wasm/keeper"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	PrefixVestingInfo         = "vesting_info"
	PrefixVestingInfoAsPrefix = util.GeneratePrefix(PrefixVestingInfo)
)

type (
	SingleVestingInfo struct {
		OwnerAddress sdk.AccAddress `json:"owner_address"`
		VestingDenom struct {
			Cw20   sdk.AccAddress `json:"cw20"`
			Native string         `json:"native"`
		} `json:"vesting_denom"`
		VestingAmount   sdk.Int `json:"vesting_amount"`
		VestedAmount    sdk.Int `json:"vested_amount"`
		VestingSchedule struct {
			StartTime       uint64 `json:"start_time,string"`
			EndTime         uint64 `json:"end_time,string"`
			VestingInterval uint64 `json:"vesting_interval,string"`
		} `json:"vesting_schedule"`
		ClaimableAmount         sdk.Int `json:"claimable_amount"`
		ClaimableStakingRewards sdk.Int `json:"claimable_staking_rewards"`
	}
)

// ExportVestingContracts look for ALL contracts that implements vesting_info key
func ExportVestingContracts(app *terra.TerraApp, contractsMap common.ContractsMap, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {

	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)

	var finalBalance util.SnapshotBalanceAggregateMap

	for contractAddr, _ := range contractsMap {
		vesting, isVesting := checkIfVesting(ctx, qs, app.WasmKeeper, contractAddr)

		// skip if not vesting
		if !isVesting {
			continue
		}

		vestingResult := handleSingleVesting(vesting)
		finalBalance = util.MergeSnapshots(finalBalance, vestingResult)
	}

	return finalBalance, nil
}

func checkIfVesting(ctx context.Context, qs wasmtypes.QueryServer, keeper wasmkeeper.Keeper, contractAddr string) (*SingleVestingInfo, bool) {
	addr, _ := sdk.AccAddressFromBech32(contractAddr)
	actx := sdk.UnwrapSDKContext(ctx)

	var singleVesting = SingleVestingInfo{}
	keeper.IterateContractStateWithPrefix(actx, addr, []byte(PrefixVestingInfo), func(_, _ []byte) bool {
		if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: addr.String(),
			QueryMsg:        []byte("{\"vesting_info\":{}}"),
		}, &singleVesting); err != nil {
			return true
		}

		return true
	})

	if !singleVesting.VestingAmount.IsNil() {
		return &singleVesting, true
	}

	return nil, false
}

func handleSingleVesting(vestingInfo *SingleVestingInfo) util.SnapshotBalanceAggregateMap {
	ownerAddress := vestingInfo.OwnerAddress
	amount := vestingInfo.VestingAmount.Sub(vestingInfo.VestedAmount)

	var denom string
	if vestingInfo.VestingDenom.Native != "" {
		denom = vestingInfo.VestingDenom.Native
	} else {
		denom = vestingInfo.VestingDenom.Cw20.String()
	}

	utilDenom, ok := coalesceToBalanceDenom(denom)
	if !ok {
		return nil
	}

	return util.SnapshotBalanceAggregateMap{
		ownerAddress.String(): {
			{
				Denom:   utilDenom,
				Balance: amount,
			},
		},
	}
}

func coalesceToBalanceDenom(assetName string) (string, bool) {
	switch assetName {
	case "uusd":
		return util.DenomUST, true
	case "uluna":
		return util.DenomLUNA, true
	case util.AddressBLUNA:
		return util.DenomBLUNA, true
	case util.AddressSTLUNA:
		return util.DenomSTLUNA, true
	case util.AddressCLUNA:
		return util.DenomCLUNA, true
	case util.AddressPLUNA:
		return util.DenomPLUNA, true
	case util.AddressNLUNA:
		return util.DenomNLUNA, true
	case util.AddressSTEAK:
		return util.DenomSTEAK, true
	case util.AddressLUNAX:
		return util.DenomLUNAX, true
	}

	return "", false
}
