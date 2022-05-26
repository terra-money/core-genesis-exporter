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
	"github.com/terra-money/core/app/export/whitewhale"
)

func ExportContracts(app *terra.TerraApp) {
	var err error

	bl := NewBlacklist()
	// snapshot := make(util.SnapshotBalanceAggregateMap)

	logger := app.Logger()
	logger.Info(fmt.Sprintf("Exporting Contracts @ %d", app.LastBlockHeight()))

	//fmt.Println(ExportSuberra(app))
	//fmt.Println(alice.ExportAlice(app, bl))
	compoundedLps, err := exportCompounders(app)
	if err != nil {
		panic(err)
	}
	snapshot, err := astroport.ExportAstroportLP(app, bl, compoundedLps)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", snapshot.SumOfDenom(util.DenomBLUNA))
	fmt.Printf("%s\n", snapshot.SumOfDenom(util.DenomSTLUNA))

	err = whitewhale.ExportWhiteWhaleVaults(app, snapshot, &bl)
	if err != nil {
		panic(err)
	}

	err = kujira.ExportKujiraVault(app, snapshot, &bl)
	if err != nil {
		panic(err)
	}

	err = lido.ExportBSTLunaHolders(app, snapshot, bl)
	if err != nil {
		panic(err)
	}
	err = lido.ExportLidoRewards(app, snapshot, bl)
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
