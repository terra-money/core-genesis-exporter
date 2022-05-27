package cw3

import (
	"encoding/json"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/generic/common"
	util "github.com/terra-money/core/app/export/util"
)

type Cw3InitMsg struct {
	Voters []Voter `json:"voters"`
}

type Voter struct {
	Address string `json:"addr"`
	Weight  int64  `json:"weight"`
}

func ExportCW3(app *terra.TerraApp, contractsMap common.ContractsMap, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)

	var finalBalance = make(util.SnapshotBalanceAggregateMap)
	for addr, ci := range contractsMap {
		var initmsg Cw3InitMsg
		if err := json.Unmarshal(ci.InitMsg, &initmsg); err != nil {
			// not a cw3 contract
			continue
		}

		if len(initmsg.Voters) == 0 {
			continue
		}

		fmt.Println(addr, initmsg.Voters, string(ci.InitMsg))

		// register this contract in blacklist map
		bl.RegisterAddress(util.DenomUST, addr)
		bl.RegisterAddress(util.DenomLUNA, addr)
		bl.RegisterAddress(util.DenomAUST, addr)
		bl.RegisterAddress(util.DenomBLUNA, addr)

		voters := initmsg.Voters

		addrr, _ := sdk.AccAddressFromBech32(addr)
		nativeBalance := app.BankKeeper.GetAllBalances(sdk.UnwrapSDKContext(ctx), addrr)
		ustBalance := nativeBalance.AmountOf("uusd")
		lunaBalance := nativeBalance.AmountOf("uluna")
		aUSTBalance, _ := util.GetCW20Balance(ctx, qs, util.AddressAUST, addr)
		bLUNABalance, _ := util.GetCW20Balance(ctx, qs, util.AddressBLUNA, addr)

		// get total weight
		var totalWeight int64
		for _, voter := range voters {
			totalWeight = totalWeight + int64(voter.Weight)
		}
		tw := sdk.NewDec(totalWeight)

		// split funds, append to final balance
		for _, voter := range voters {
			w := sdk.NewDec(int64(voter.Weight))
			finalBalance.AppendOrAddBalance(voter.Address, util.SnapshotBalance{
				Denom:   util.DenomUST,
				Balance: sdk.NewDecFromInt(ustBalance).Mul(w).Quo(tw).TruncateInt(),
			})
			finalBalance.AppendOrAddBalance(voter.Address, util.SnapshotBalance{
				Denom:   util.DenomLUNA,
				Balance: sdk.NewDecFromInt(lunaBalance).Mul(w).Quo(tw).TruncateInt(),
			})
			finalBalance.AppendOrAddBalance(voter.Address, util.SnapshotBalance{
				Denom:   util.DenomAUST,
				Balance: sdk.NewDecFromInt(aUSTBalance).Mul(w).Quo(tw).TruncateInt(),
			})
			finalBalance.AppendOrAddBalance(voter.Address, util.SnapshotBalance{
				Denom:   util.DenomBLUNA,
				Balance: sdk.NewDecFromInt(bLUNABalance).Mul(w).Quo(tw).TruncateInt(),
			})
		}
	}

	return finalBalance, nil
}
