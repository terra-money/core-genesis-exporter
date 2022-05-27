package ink

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	// stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"

	// wasmkeeper "github.com/terra-money/core/x/wasm/keeper"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

const (
	InkVault     = "terra1v579mvp2xxw3st7glgaurfla5pxses0jdwedde"
	InkAUstVault = "terra1lhavzcmdh6073j82m4sa8n7w5t5c4prugzl68a"
	InkParty     = "terra1p4y54kfdn9uhvh62rjvgz3sydceuw9s6c65aef"
	InkCore      = "terra1nlsfl8djet3z70xu2cj7s9dn7kzyzzfz5z2sd9"
)

type PartyRes struct {
	Parties []PartyInfo `json:"parties"`
}

type PartyInfo struct {
	Info struct {
		Id        int    `json:"id"`
		PartyAddr string `json:"party_addr"`
	} `json:"info"`
	Deposits []struct {
		Amount  sdk.Int `json:"amount"`
		Address string  `json:"address"`
	} `json:"deposits"`
}

type VaultRes struct {
	Vaults []Vault `json:"vaults"`
}

type Vault struct {
	Address           string  `json:"address"`
	VaultAddr         string  `json:"vault_addr"`
	UstAmountInAnchor sdk.Int `json:"initial_anchor"`
	UstAmountInCore   sdk.Int `json:"initial_core"`
}

// Ink protocol
// Depending on how users deposit and their configuration
// UST is deposited into a few contracts
// 1. InkAustVault (Party goes into here too)
// 2. Individual interest vaults per user (InkVault)
func ExportContract(
	app *terra.TerraApp,
	bl util.Blacklist,
) (util.SnapshotBalanceAggregateMap, error) {
	app.Logger().Info("Exporting Ink Protocol")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)
	deposits, err := getAllDeposits(ctx, q)
	if err != nil {
		return nil, err
	}

	partyInfos, err := getAllParties(ctx, q)
	if err != nil {
		return nil, err
	}

	vaults, err := getAllVaults(ctx, q)
	if err != nil {
		return nil, err
	}

	for _, party := range partyInfos {
		if !deposits[party.Info.PartyAddr].IsNil() {
			deposits[party.Info.PartyAddr] = sdk.NewInt(0)
		}
		sum := sdk.NewInt(0)
		for _, dp := range party.Deposits {
			sum = sum.Add(dp.Amount)
			if deposits[dp.Address].IsNil() {
				deposits[dp.Address] = dp.Amount
			} else {
				deposits[dp.Address] = deposits[dp.Address].Add(dp.Amount)
			}
		}
	}

	for _, vault := range vaults {
		bl.RegisterAddress(util.DenomAUST, vault.VaultAddr)
		if !deposits[vault.VaultAddr].IsNil() {
			deposits[vault.VaultAddr] = sdk.NewInt(0)
		}
		if deposits[vault.Address].IsNil() {
			deposits[vault.Address] = vault.UstAmountInAnchor.Add(vault.UstAmountInCore)
		} else {
			deposits[vault.Address] = deposits[vault.Address].Add(vault.UstAmountInAnchor.Add(vault.UstAmountInCore))
		}
	}

	totalAUstLocked, err := getTotalAUstLocked(ctx, q, vaults)
	if err != nil {
		return nil, err
	}
	totalDeposits := util.Sum(deposits)

	snapshot := make(util.SnapshotBalanceAggregateMap)
	for addr, amount := range deposits {
		aUstBalance := amount.Mul(totalAUstLocked).Quo(totalDeposits)
		snapshot[addr] = append(snapshot[addr], util.SnapshotBalance{
			Denom:   util.DenomAUST,
			Balance: aUstBalance,
		})
	}
	bl.RegisterAddress(util.DenomAUST, InkAUstVault)
	return snapshot, nil
}

func getAllDeposits(ctx context.Context, q wasmtypes.QueryServer) (map[string]sdk.Int, error) {
	var getPlayers func(string) error
	limit := 500

	deposits := make(map[string]sdk.Int)
	getPlayers = func(startAddress string) error {
		var PlayersRes struct {
			Players []struct {
				Address string  `json:"address"`
				Amount  sdk.Int `json:"amount"`
			} `json:"players"`
		}
		err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: InkCore,
			QueryMsg:        []byte(playerQuery(limit, startAddress, 0)),
		}, &PlayersRes)
		if err != nil {
			return err
		}
		var lastPlayer string
		for _, player := range PlayersRes.Players {
			lastPlayer = player.Address
			if deposits[player.Address].IsNil() {
				deposits[player.Address] = player.Amount
			} else {
				deposits[player.Address] = deposits[player.Address].Add(player.Amount)
			}
		}
		if len(PlayersRes.Players) < limit {
			return nil
		}
		return getPlayers(lastPlayer)
	}
	err := getPlayers("")
	if err != nil {
		return nil, err
	}
	return deposits, nil
}

func getAllParties(ctx context.Context, q wasmtypes.QueryServer) ([]PartyInfo, error) {
	var getParties func(startAfter int) error
	limit := 80

	var partyInfos []PartyInfo
	var partyRes PartyRes
	getParties = func(startAfter int) error {
		err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
			ContractAddress: InkParty,
			QueryMsg:        []byte(fmt.Sprintf("{\"parties_with_deposits\": {\"limit\": %d, \"start_after\": %d}}", limit, startAfter)),
		}, &partyRes)
		if err != nil {
			return err
		}

		partyInfos = append(partyInfos, partyRes.Parties...)
		if len(partyRes.Parties) < limit {
			return nil
		}
		return getParties(startAfter + limit)
	}
	err := getParties(0)
	if err != nil {
		return nil, err
	}
	return partyInfos, nil
}

func getAllVaults(ctx context.Context, q wasmtypes.QueryServer) ([]Vault, error) {
	var vaultRes VaultRes
	err := util.ContractQuery(ctx, q, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: InkVault,
		// there are only ~1053 (incl. empty vaults)
		QueryMsg: []byte("{\"vault_deposits\": {\"limit\": 1000, \"include_zero_deposit\": false}}"),
	}, &vaultRes)
	if err != nil {
		return nil, err
	}
	return vaultRes.Vaults, nil
}

func playerQuery(limit int, startAddress string, sid int) string {
	if startAddress != "" {
		return fmt.Sprintf("{\"players\":{\"sid\":%d, \"limit\": %d, \"start_address\": \"%s\"}}", sid, limit, startAddress)
	} else {
		return fmt.Sprintf("{\"players\":{\"sid\":%d, \"limit\": %d}}", sid, limit)
	}
}

func getTotalAUstLocked(ctx context.Context, q wasmtypes.QueryServer, vaults []Vault) (sdk.Int, error) {
	sum := sdk.NewInt(0)
	for _, v := range vaults {
		balance, err := util.GetCW20Balance(ctx, q, util.AUST, v.VaultAddr)
		if err != nil {
			return sdk.Int{}, err
		}
		sum = sum.Add(balance)
	}
	balance, err := util.GetCW20Balance(ctx, q, util.AUST, InkAUstVault)
	if err != nil {
		return sdk.Int{}, err
	}
	sum = sum.Add(balance)
	return sum, nil
}
