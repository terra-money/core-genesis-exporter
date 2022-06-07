package cw3

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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

// Export CW3 and other treasuries
// For genesis snapshot, we split CW3 holdings for UST, aUST LUNA to all voters
// We missed other staking derivatives, LP and lockdrop holdings
// For the airdrop fix, we will index everything and remove what we have already airdropped
func ExportCW3(app *terra.TerraApp, contractsMap common.ContractsMap, snapshot util.SnapshotBalanceAggregateMap, bl util.Blacklist) error {
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)

	contractBalanceMap := make(map[string]map[string]sdk.Int)
	for addr, ci := range contractsMap {
		var initmsg Cw3InitMsg
		if err := json.Unmarshal(ci.InitMsg, &initmsg); err != nil {
			// not a cw3 contract
			continue
		}

		if len(initmsg.Voters) == 0 {
			continue
		}

		// register this contract in blacklist map
		bl.RegisterAddress(util.DenomUST, addr)
		bl.RegisterAddress(util.DenomLUNA, addr)
		bl.RegisterAddress(util.DenomAUST, addr)

		voters := initmsg.Voters

		addrr, _ := sdk.AccAddressFromBech32(addr)
		nativeBalance := app.BankKeeper.GetAllBalances(sdk.UnwrapSDKContext(ctx), addrr)
		ustBalance := nativeBalance.AmountOf("uusd")
		lunaBalance := nativeBalance.AmountOf("uluna")
		aUSTBalance, _ := util.GetCW20Balance(ctx, qs, util.AddressAUST, addr)

		// This is used to remove from the final snapshot since we already allocated it
		contractBalanceMap[addr] = make(map[string]sdk.Int)
		contractBalanceMap[addr][util.DenomUST] = ustBalance
		contractBalanceMap[addr][util.DenomLUNA] = lunaBalance
		contractBalanceMap[addr][util.DenomAUST] = aUSTBalance

		// get total weight
		var totalWeight int64
		for _, voter := range voters {
			totalWeight = totalWeight + int64(voter.Weight)
		}
		tw := sdk.NewDec(totalWeight)

		// split funds, append to final balance
		for _, voter := range voters {
			w := sdk.NewDec(int64(voter.Weight))
			snapshot.AppendOrAddBalance(voter.Address, util.SnapshotBalance{
				Denom:   util.DenomUST,
				Balance: sdk.NewDecFromInt(ustBalance).Mul(w).Quo(tw).TruncateInt(),
			})
			snapshot.AppendOrAddBalance(voter.Address, util.SnapshotBalance{
				Denom:   util.DenomLUNA,
				Balance: sdk.NewDecFromInt(lunaBalance).Mul(w).Quo(tw).TruncateInt(),
			})
			snapshot.AppendOrAddBalance(voter.Address, util.SnapshotBalance{
				Denom:   util.DenomAUST,
				Balance: sdk.NewDecFromInt(aUSTBalance).Mul(w).Quo(tw).TruncateInt(),
			})
		}
	}

	// Subtract what has already been airdropped
	for addr, balances := range contractBalanceMap {
		for i, sbs := range snapshot[addr] {
			if !balances[sbs.Denom].IsNil() {
				remaining := sbs.Balance.Sub(balances[sbs.Denom])
				if remaining.IsNegative() {
					panic(fmt.Errorf("negative balance %s, %s, %s", addr, sbs.Denom, remaining))
				}

				snapshot[addr][i] = util.SnapshotBalance{
					Denom:   sbs.Denom,
					Balance: remaining,
				}
			}
		}
	}
	mapKnownContracts(snapshot)
	return nil
}

const contractMappingFile = "./app/export/generic/common/contract-mapping.csv"

func mapKnownContracts(snapshot util.SnapshotBalanceAggregateMap) {
	file, err := os.Open(contractMappingFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// For contracts that we know, assign it to the right owner
	var cAdd string
	var rAdd string
	for scanner.Scan() {
		adds := strings.Split(scanner.Text(), ",")
		// Validate addresses
		add, err := sdk.AccAddressFromBech32(adds[0])
		if err == nil {
			cAdd = add.String()
		}
		add, err = sdk.AccAddressFromBech32(adds[1])
		if err == nil {
			rAdd = add.String()
		}
	}
	for _, b := range snapshot[cAdd] {
		snapshot.AppendOrAddBalance(rAdd, b)
	}
	delete(snapshot, cAdd)
}
