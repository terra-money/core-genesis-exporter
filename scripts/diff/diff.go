package main

import (
	"encoding/json"
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

type Genesis struct {
	AppState struct {
		Bank Bank `json:"bank"`
	} `json:"app_state"`
}

type Bank struct {
	Balances Balances `json:"balances"`
}

type Balances []types.Balance

func main() {
	args := os.Args[1:]
	aGenesisPath := args[0]
	bGenesisPath := args[1]

	aGenesis, err := parseGenesis(aGenesisPath)
	if err != nil {
		panic(err)
	}
	bGenesis, err := parseGenesis(bGenesisPath)
	if err != nil {
		panic(err)
	}

	fmt.Printf("lengths - a: %d, b: %d\n", len(aGenesis.AppState.Bank.Balances), len(bGenesis.AppState.Bank.Balances))

	balance, err := checkDiff(aGenesis.AppState.Bank.Balances, bGenesis.AppState.Bank.Balances)
	if err != nil {
		panic(err)
	}

	output, err := json.MarshalIndent(balance, "", "  ")
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("./output.json", output, 0700)
	if err != nil {
		panic(err)
	}

	printBalanceSummary(balance)
}

func parseGenesis(path string) (Genesis, error) {
	var genesis Genesis
	genesisBytes, err := os.ReadFile(path)
	if err != nil {
		return genesis, err
	}
	err = json.Unmarshal(genesisBytes, &genesis)
	if err != nil {
		return genesis, err
	}
	return genesis, nil
}

func checkDiff(aB, bG Balances) (Balances, error) {
	aMap := make(map[string]map[string]sdk.Int)
	for _, b := range aB {
		if aMap[b.Address] == nil {
			aMap[b.Address] = make(map[string]sdk.Int)
		}
		for _, c := range b.Coins {
			if aMap[b.Address][c.Denom].IsNil() {
				aMap[b.Address][c.Denom] = sdk.NewInt(0)
			}
			aMap[b.Address][c.Denom] = aMap[b.Address][c.Denom].Add(c.Amount)
		}
	}
	var newBalance Balances
	for _, b := range bG {
		nB := types.Balance{
			Address: b.Address,
			Coins:   sdk.Coins{},
		}
		for _, c := range b.Coins {
			oldValue := aMap[b.Address][c.Denom]
			if oldValue.IsNil() {
				oldValue = sdk.NewInt(0)
			}
			diffValue := c.Amount.Sub(oldValue)
			if diffValue.GT(sdk.NewInt(100000)) {
				coin := sdk.Coin{
					Denom:  c.Denom,
					Amount: diffValue,
				}
				nB.Coins = append(nB.Coins, coin)
			}
		}
		if len(nB.Coins) > 0 {
			newBalance = append(newBalance, nB)
		}
	}
	return newBalance, nil
}

func printBalanceSummary(bals Balances) {
	denomMap := make(map[string]sdk.Int)
	for _, b := range bals {
		for _, c := range b.Coins {
			if denomMap[c.Denom].IsNil() {
				denomMap[c.Denom] = sdk.NewInt(0)
			}
			denomMap[c.Denom] = denomMap[c.Denom].Add(c.Amount)
		}
	}
	fmt.Printf("%v\n", denomMap)
}
