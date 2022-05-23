package app

import (
	"fmt"
)

func ExportContracts(app *TerraApp) {

	bl := NewBlacklist()

	logger := app.Logger()
	logger.Info(fmt.Sprintf("Exporting Contracts @ %d", app.LastBlockHeight()))

	//fmt.Println(ExportSuberra(app))
	fmt.Println(ExportAlice(app, bl))
}

type blacklist map[string][]string

func NewBlacklist() blacklist {
	return blacklist{
		DenomUST:  []string{},
		DenomLUNA: []string{},
		DenomAUST: []string{},
	}
}

func (bl blacklist) RegisterAddress(denom string, address string) {
	bl[denom] = append(bl[denom], address)
}