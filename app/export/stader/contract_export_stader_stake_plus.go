package stader

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	wasmtypes "github.com/terra-money/core/x/wasm/types"

	"github.com/terra-money/core/app/export/util"
)

const (
	StakeRegistry = "terra1ku85smu4ews088g64sk8wjx5edv8m42205ympl"
)

type UserUndelegationRequest struct {
	BatchId int     `json:"batch_id"`
	Shares  sdk.Dec `json:"shares"`
	Amount  sdk.Int `json:"amount"`
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

var StaderFundsContracts = []string{
	"terra1smgqsx87cd9q62pa6mrvmydayxd2jegys3cd2d",
	"terra167szfqgnqpezer5tfzf9f0uqj3lw6t59y2f3ej",
	"terra1wtxc4vfk8r9rdullaqm5euxvqs3javdkyy0pz9",
}

// ExportStakePlus Export staked Luna balances for users.
func ExportStakePlus(app *terra.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	snapshot := make(util.SnapshotBalanceAggregateMap)

	logger := app.Logger()
	logger.Info("Exporting Stader Stake+ balances")

	// get all stake+ contracts from registry
	var staderContracts struct {
		Contracts []string `json:"contracts"`
	}
	if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: StakeRegistry,
		QueryMsg:        []byte("{\"get_staking_contracts\": {}}"),
	}, &staderContracts); err != nil {
		return nil, err
	}

	for _, contract := range staderContracts.Contracts {
		// Exclude Fund contracts which don't have get_all_users queries.
		if contains(StaderFundsContracts, contract) {
			continue
		}

		var stakePlusUsers struct {
			UserInfo []struct {
				UserAddr    string  `json:"user_addr"`
				TotalShares sdk.Dec `json:"total_shares"`
				TotalAmount struct {
					Denom  string  `json:"denom"`
					Amount sdk.Int `json:"amount"`
				} `json:"total_amount"`
			} `json:"user_info"`
		}

		var offset = ""
		for {
			query := "{\"get_all_users\": {\"limit\": 30}}"
			if offset != "" {
				query = fmt.Sprintf("{\"get_all_users\": {\"start_after\": \"%s\", \"limit\": 30}}", offset)
			}

			if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
				ContractAddress: contract,
				QueryMsg:        []byte(query),
			}, &stakePlusUsers); err != nil {
				return nil, err
			}

			if len(stakePlusUsers.UserInfo) == 0 {
				break
			}

			for _, userInfo := range stakePlusUsers.UserInfo {
				snapshot.AppendOrAddBalance(userInfo.UserAddr, util.SnapshotBalance{
					Denom:   util.DenomLUNA,
					Balance: userInfo.TotalAmount.Amount,
				})

				// Fetch undelegation requests for this user.
				undelegations, err := getUserUndelegations(ctx, q, contract, userInfo.UserAddr)
				if err != nil {
					return nil, err
				}

				// Add undelegations to users total.
				for _, undelegation := range undelegations {
					snapshot.AppendOrAddBalance(userInfo.UserAddr, util.SnapshotBalance{
						Denom:   util.DenomLUNA,
						Balance: undelegation.Amount,
					})
				}
			}

			offset = stakePlusUsers.UserInfo[len(stakePlusUsers.UserInfo)-1].UserAddr
		}
	}

	return snapshot, nil
}

// getUserUndelegations fetch all user undelegation requests.
func getUserUndelegations(ctx context.Context, q wasmtypes.QueryServer, contract string, userAddr string) ([]UserUndelegationRequest, error) {
	undelegationRequests := []UserUndelegationRequest{}
	var offset = -1
	for {
		query := fmt.Sprintf("{\"get_user_undelegation_records\":{\"limit\": 30,\"user_addr\":\"%s\"}}", userAddr)
		if offset != -1 {
			query = fmt.Sprintf("{\"get_user_undelegation_records\":{\"start_after\":%d,\"limit\": 30,\"user_addr\":\"%s\"}}", offset, userAddr)
		}

		var stakePlusUndelegations []UserUndelegationRequest

		if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: contract,
			QueryMsg:        []byte(query),
		}, &stakePlusUndelegations); err != nil {
			panic(err)
		}

		if len(stakePlusUndelegations) == 0 {
			break
		}

		undelegationRequests = append(undelegationRequests, stakePlusUndelegations...)

		offset = int(stakePlusUndelegations[len(stakePlusUndelegations)-1].BatchId)
	}

	return undelegationRequests, nil
}
