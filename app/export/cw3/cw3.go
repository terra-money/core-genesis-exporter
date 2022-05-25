package cw3

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

type NativeMultiSig struct {
	PubKey  string
	Signers []string
}

type Cw3InitMsg struct {
	Voters []Voter `json:"voters"`
}

type Voter struct {
	Address string `json:"addr"`
	Weight  int    `json:"weight"`
}

func SplitFundsToVoters(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap, b *util.Blacklist) error {
	app.Logger().Info("Splitting funds from CW3 contracts")
	q := util.PrepWasmQueryServer(app)
	cw3s, err := getCw3Voters(app, q)
	if err != nil {
		return err
	}
	totalUstBalance := snapshot.SumOfDenom(util.DenomUST)
	for contract, voters := range cw3s {
		totalWeight := 0
		for _, v := range voters {
			totalWeight += v.Weight
		}
		balances, ok := snapshot[contract]
		if ok {
			for i, sb := range balances {
				balances[i] = util.SnapshotBalance{
					Denom:   sb.Denom,
					Balance: sdk.NewInt(0),
				}

				for _, v := range voters {
					snapshot[v.Address] = append(snapshot[v.Address], util.SnapshotBalance{
						Denom:   sb.Denom,
						Balance: sb.Balance.MulRaw(int64(v.Weight)).QuoRaw(int64(totalWeight)),
					})
				}
			}
		}
	}
	err = util.AlmostEqual("uust balance miss match after vote split", totalUstBalance, snapshot.SumOfDenom(util.DenomUST), sdk.NewInt(100000))
	if err != nil {
		return err
	}
	return nil
}

func getCw3Voters(app *terra.TerraApp, q wasmtypes.QueryServer) (map[string][]Voter, error) {
	app.Logger().Info("... Finding CW3 Contracts")
	ctx := util.PrepCtx(app)
	cw3ToMultisig := make(map[string][]Voter)
	totalNumberOfSeenContracts := 0
	app.WasmKeeper.IterateContractInfo(types.UnwrapSDKContext(ctx), func(ci wasmtypes.ContractInfo) bool {
		totalNumberOfSeenContracts += 1
		if totalNumberOfSeenContracts%50 == 0 {
			fmt.Printf("\r%d", totalNumberOfSeenContracts)
		}
		var cw3InitMsg Cw3InitMsg
		err := json.Unmarshal(ci.InitMsg, &cw3InitMsg)
		if err != nil {
			return false
		}
		if len(cw3InitMsg.Voters) > 0 {
			cw3ToMultisig[ci.Address] = cw3InitMsg.Voters
		}
		return false
	})
	return cw3ToMultisig, nil
}

func Test(app *terra.TerraApp) error {
	ctx := util.PrepCtx(app)
	testCw3 := "terra1hxrd8pnqytqpelape3aemprw3a023wryw7p0xn"
	balance, err := util.GetNativeBalance(ctx, app.BankKeeper, util.DenomUST, testCw3)
	if err != nil {
		panic(err)
	}
	snapshot := make(util.SnapshotBalanceAggregateMap)
	snapshot.AppendOrAddBalance(testCw3, util.SnapshotBalance{
		Denom:   util.DenomUST,
		Balance: balance,
	})

	bl := make(util.Blacklist)
	err = SplitFundsToVoters(app, snapshot, &bl)
	if err != nil {
		panic(err)
	}
	return nil
}
