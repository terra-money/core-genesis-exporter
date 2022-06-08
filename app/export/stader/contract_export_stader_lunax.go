package stader

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	LunaX      = "terra17y9qkl8dfkeg4py7n0g5407emqnemc3yqk5rup"
	LunaXState = "terra1xacqx447msqp46qmv8k2sq6v5jh9fdj37az898"
)

type LunaXUndelegationRequest struct {
	BatchId     int     `json:"batch_id"`
	TokenAmount sdk.Int `json:"token_amount"`
}

// ExportLunaX get Luna balance for all accounts, multiply ER
func ExportLunaX(app *terra.TerraApp, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	snapshot := make(util.SnapshotBalanceAggregateMap)

	logger := app.Logger()
	logger.Info("Exporting LunaX holders")

	var lunaxBalances = make(util.BalanceMap)
	if err := util.GetCW20AccountsAndBalances(ctx, app.WasmKeeper, LunaX, lunaxBalances); err != nil {
		return nil, err
	}

	exchangeRate, err := GetLunaXExchangeRate(ctx, q)
	if err != nil {
		return nil, err
	}

	// balance * ER
	for address, balance := range lunaxBalances {
		if !balance.IsZero() {
			snapshot[address] = append(snapshot[address], util.SnapshotBalance{
				Denom:   util.DenomLUNA,
				Balance: exchangeRate.MulInt(balance).TruncateInt(),
			})
		}

		// Fetch undelegation requests for this user.
		undelegations, err := GetLunaXUndelegations(ctx, q, LunaXState, address)
		if err != nil {
			return nil, err
		}

		for _, undelegation := range undelegations {
			if undelegation.TokenAmount.IsZero() {
				continue
			}

			snapshot.AppendOrAddBalance(address, util.SnapshotBalance{
				Denom:   util.DenomLUNA,
				Balance: exchangeRate.MulInt(undelegation.TokenAmount).TruncateInt(),
			})
		}
	}

	bl.RegisterAddress(util.DenomLUNA, LunaXState)
	return snapshot, nil
}

// GetLunaXExchangeRate Get the exchange rate from LunaX to Luna.
func GetLunaXExchangeRate(ctx context.Context, q wasmtypes.QueryServer) (sdk.Dec, error) {
	// get LunaX <> Luna ER
	var lunaxStateResponse struct {
		State struct {
			ExchangeRate sdk.Dec `json:"exchange_rate"`
		} `json:"state"`
	}

	if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: LunaXState,
		QueryMsg:        []byte("{\"state\":{}}"),
	}, &lunaxStateResponse); err != nil {
		return sdk.NewDec(0), err
	}

	return lunaxStateResponse.State.ExchangeRate, nil
}

func ResolveToLuna(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)

	er, err := GetLunaXExchangeRate(ctx, qs)
	if err != nil {
		return fmt.Errorf("error fetching LunaX <> Luna ER", err)
	}

	for _, sbs := range snapshot {
		for i, sb := range sbs {
			if sb.Denom == util.DenomLUNAX {
				sbs[i] = util.SnapshotBalance{
					Denom:   util.DenomLUNA,
					Balance: er.MulInt(sb.Balance).TruncateInt(),
				}
			}
		}
	}

	return nil
}

// GetLunaXUndelegations fetch all user undelegation requests.
func GetLunaXUndelegations(ctx context.Context, q wasmtypes.QueryServer, contract string, userAddr string) ([]LunaXUndelegationRequest, error) {
	undelegationRequests := []LunaXUndelegationRequest{}
	var offset = -1
	for {
		query := fmt.Sprintf("{\"get_user_undelegation_records\":{\"limit\": 30,\"user_addr\":\"%s\"}}", userAddr)
		if offset != -1 {
			query = fmt.Sprintf("{\"get_user_undelegation_records\":{\"start_after\":%d,\"limit\": 30,\"user_addr\":\"%s\"}}", offset, userAddr)
		}

		var lunaXUndelegations []LunaXUndelegationRequest

		if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: contract,
			QueryMsg:        []byte(query),
		}, &lunaXUndelegations); err != nil {
			panic(err)
		}

		if len(lunaXUndelegations) == 0 {
			break
		}

		undelegationRequests = append(undelegationRequests, lunaXUndelegations...)
		offset = int(lunaXUndelegations[len(lunaXUndelegations)-1].BatchId)
	}

	return undelegationRequests, nil
}

func Audit(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap) error {
	app.Logger().Info("Audit -- LunaX")
	ctx := util.PrepCtx(app)

	lunaBalance, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomLUNA, LunaXState)
	if err != nil {
		return err
	}

	// TODO: Need to also query staked Luna.
	if err := util.AlmostEqual(util.DenomLUNA, lunaBalance, snapshot.SumOfDenom(util.DenomLUNA), sdk.NewInt(1000000)); err != nil {
		return err
	}

	return nil
}
