package app

import (
	"fmt"

	terra "github.com/terra-money/core/app"

	"github.com/terra-money/core/app/export/edge"
	"github.com/terra-money/core/app/export/prism"
	"github.com/terra-money/core/app/export/util"
)

func ExportContracts(app *terra.TerraApp) {

	snapshot := make(util.SnapshotBalanceAggregateMap)
	bl := NewBlacklist()

	logger := app.Logger()
	logger.Info(fmt.Sprintf("Exporting Contracts @ %d", app.LastBlockHeight()))

	//fmt.Println(ExportSuberra(app))
	// fmt.Println(kujira.ExportKujiraStaking(app, &bl))
	// fmt.Println(alice.ExportAlice(app, bl))
	// lido.ExportLidoContract(app, make(map[string]types.Int), make(map[string]types.Int), &bl)
	// ink.ExportContract(app, &bl)
	err := prism.ExportLimitOrderContract(app, snapshot, &bl)
	if err != nil {
		panic(err)
	}
	err = edge.ExportContract(app, snapshot, &bl)
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
