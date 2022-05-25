package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/astroport"
	"github.com/terra-money/core/app/export/kujira"
	"github.com/terra-money/core/app/export/lido"
	"github.com/terra-money/core/app/export/mars"
	"github.com/terra-money/core/app/export/prism"
	"github.com/terra-money/core/app/export/util"
)

func ExportContracts(app *terra.TerraApp) {
	var err error

	snapshot := make(util.SnapshotBalanceAggregateMap)
	bl := NewBlacklist()

	logger := app.Logger()
	logger.Info(fmt.Sprintf("Exporting Contracts @ %d", app.LastBlockHeight()))

	err = kujira.ExportKujiraVault(app, snapshot, &bl)
	if err != nil {
		panic(err)
	}

	//fmt.Println(ExportSuberra(app))
	//fmt.Println(alice.ExportAlice(app, bl))
	compoundedLps, err := exportCompounders(app)
	if err != nil {
		panic(err)
	}
	astroport.ExportAstroportLP(app, bl, compoundedLps)
	// fmt.Println(kujira.ExportKujiraStaking(app, &bl))
	// fmt.Println(alice.ExportAlice(app, bl))
	// err = edge.ExportContract(app, snapshot, &bl)
	// if err != nil {
	// 	panic(err)
	// }

	err = lido.ExportBSTLunaHolders(app, snapshot, &bl)
	if err != nil {
		panic(err)
	}
	err = lido.ExportLidoRewards(app, snapshot, &bl)
	if err != nil {
		panic(err)
	}
	err = lido.ResolveLidoLuna(app, snapshot, bl)
	if err != nil {
		panic(err)
	}

	// ink.ExportContract(app, &bl)
	// err = aperture.ExportApertureVaults(app, util.Snapshot(util.PreAttack), snapshot, &bl)
	// if err != nil {
	// 	panic(err)
	// }

	err = prism.ExportLimitOrderContract(app, snapshot, &bl)
	if err != nil {
		panic(err)
	}
	err = prism.ExportContract(app, snapshot, &bl)
	if err != nil {
		panic(err)
	}

	err = prism.ResolveToLuna(app, snapshot, bl)
	if err != nil {
		panic(err)
	}
}

func NewBlacklist() util.Blacklist {
	return util.Blacklist{
		util.DenomUST:  []string{},
		util.DenomLUNA: []string{},
		util.DenomAUST: []string{},
	}
}

func exportCompounders(app *terra.TerraApp) (map[string]map[string]map[string]sdk.Int, error) {
	finalMap := make(map[string]map[string]map[string]sdk.Int)
	// specLps, err := spectrum.ExportSpecVaultLPs(app)
	// if err != nil {
	// 	return nil, err
	// }
	// for k, v := range specLps {
	// 	finalMap[k] = v
	// }
	// apolloLps, err := apollo.ExportApolloVaultLPs(app)
	// if err != nil {
	// 	return nil, err
	// }
	// for k, v := range apolloLps {
	// 	finalMap[k] = v
	// }
	marsLps, err := mars.ExportFieldOfMarsLpTokens(app)
	if err != nil {
		return nil, err
	}
	for k, v := range marsLps {
		finalMap[k] = v
	}
	return finalMap, nil
}
