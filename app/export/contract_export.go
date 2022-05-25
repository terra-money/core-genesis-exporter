package app

import (
	"fmt"
	"github.com/terra-money/core/app/export/astroport"

	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
)

func ExportContracts(app *terra.TerraApp) {

	bl := NewBlacklist()

	logger := app.Logger()
	logger.Info(fmt.Sprintf("Exporting Contracts @ %d", app.LastBlockHeight()))

	//fmt.Println(ExportSuberra(app))
	//fmt.Println(alice.ExportAlice(app, bl))
	fmt.Println(astroport.ExportAstroportLP(app, bl))
}

func NewBlacklist() util.Blacklist {
	return util.Blacklist{
		util.DenomUST:  []string{},
		util.DenomLUNA: []string{},
		util.DenomAUST: []string{},
	}
}
