package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/apollo"
	"github.com/terra-money/core/app/export/lido"
	"github.com/terra-money/core/app/export/prism"
	"github.com/terra-money/core/app/export/spectrum"
	"github.com/terra-money/core/app/export/util"
)

func ExportContracts(app *terra.TerraApp) {
	var err error

	snapshot := make(util.SnapshotBalanceAggregateMap)
	bl := NewBlacklist()

	logger := app.Logger()
	logger.Info(fmt.Sprintf("Exporting Contracts @ %d", app.LastBlockHeight()))

	//fmt.Println(ExportSuberra(app))
	//fmt.Println(alice.ExportAlice(app, bl))
	// compoundedLps, err := exportCompounders(app)
	// astroport.ExportAstroportLP(app, bl, compoundedLps)
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
	specLps, err := spectrum.ExportSpecVaultLPs(app)
	if err != nil {
		return nil, err
	}
	apolloLps, err := apollo.ExportApolloVaultLPs(app)
	if err != nil {
		return nil, err
	}
	for k, v := range apolloLps {
		specLps[k] = v
	}
	return specLps, nil
}
