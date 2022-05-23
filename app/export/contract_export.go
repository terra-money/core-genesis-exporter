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
<<<<<<< HEAD
	// fmt.Println(alice.ExportAlice(app, bl))
	// kujira.ExportKujiraVault(app, &bl)
	kujira.ExportKujiraStaking(app, &bl)
=======
	fmt.Println(alice.ExportAlice(app, bl))
>>>>>>> c296cc37479287fd800cce0117af7281625ddd7d
}

func NewBlacklist() util.Blacklist {
	return util.Blacklist{
<<<<<<< HEAD
		util.DenomUST:  []string{},
		util.DenomLUNA: []string{},
		util.DenomAUST: []string{},
=======
		DenomUST:  []string{},
		DenomLUNA: []string{},
		DenomAUST: []string{},
>>>>>>> c296cc37479287fd800cce0117af7281625ddd7d
	}
}
