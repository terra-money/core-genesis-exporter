package stader

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	wasmtypes "github.com/terra-money/core/x/wasm/types"

	"github.com/terra-money/core/app/export/util"
)

const (
	StaderPools      = "terra1r2vv8cyt0scyxymktyfuudqs3lgtypk72w6m3m"
	StaderDelegator  = "terra1t9ree3ftvgr70fvm6y67zsqxjms8jju8kwcsdu"
	StaderLunaX      = "terra17y9qkl8dfkeg4py7n0g5407emqnemc3yqk5rup"
	StaderController = "terra1xacqx447msqp46qmv8k2sq6v5jh9fdj37az898"
	StaderSCC        = "terra127vwnwgwdvq94ce4ws76ddh0c699jt40dznrn2"
)

func ExportStaderPools(app *terra.TerraApp, bl *util.Blacklist) (util.SnapshotBalanceMap, error) {
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	// Pull users from user_registry map.
	// pub const USER_REGISTRY: Map<(&Addr, U64Key), UserPoolInfo> = Map::new("user_registry");
	prefix := util.GeneratePrefix("user_registry")
	delegatorAddr, err := sdk.AccAddressFromBech32(StaderPools)
	if err != nil {
		return nil, err
	}

	users := []string{}
	app.WasmKeeper.IterateContractStateWithPrefix(sdk.UnwrapSDKContext(ctx), delegatorAddr, prefix, func(key, value []byte) bool {
		// First character in key is a comma.
		stakerAddr := sdk.AccAddress(strings.TrimLeft(string(key), ","))

		users = append(users, stakerAddr.String())
		return false
	})

	balances := make(util.SnapshotBalanceMap)
	for _, address := range users {
		// fmt.Println(address)

		for i := 1; i < 3; i++ {
			var poolUserInfo struct {
				Info struct {
					Deposit struct {
						Staked sdk.Int `json:"staked"`
					} `json:"deposit"`
				} `json:"info"`
			}

			// TODO: Figure out why none of the addresses in the user_registry have a balance.
			if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
				ContractAddress: StaderPools,
				QueryMsg:        []byte(fmt.Sprintf("{\"get_user_computed_info\": {\"user_addr\": \"%s\", \"pool_id\": %d}}", address, i)),
			}, &poolUserInfo); err != nil {
				panic(err)
			}

			fmt.Println(poolUserInfo)
		}
	}

	// get LunaX <> LUNA ER
	var staderState struct {
		State struct {
			ExchangeRate sdk.Dec `json:"exchange_rate"`
		} `json:"state"`
	}
	if err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: StaderController,
		QueryMsg:        []byte("{\"state\":{}}"),
	}, &staderState); err != nil {
		return nil, err
	}

	// TODO: call get_user on SCC to cover pending rewards.

	bl.RegisterAddress(util.DenomLUNA, StaderPools)
	return balances, nil
}
