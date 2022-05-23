package app

import (
	"fmt"

	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/contract_export/alice"
	"github.com/terra-money/core/app/export/util"
)

func ExportContracts(app *terra.TerraApp) {

	bl := NewBlacklist()

	logger := app.Logger()
	logger.Info(fmt.Sprintf("Exporting Contracts @ %d", app.LastBlockHeight()))

	//fmt.Println(ExportSuberra(app))
	fmt.Println(alice.ExportAlice(app, bl))
}

func NewBlacklist() util.Blacklist {
	return util.Blacklist{
		DenomUST:  []string{},
		DenomLUNA: []string{},
		DenomAUST: []string{},
	}
}
