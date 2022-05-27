package util

import "github.com/terra-money/core/app/export/generic/common"

// bridge addresses
var contractWhitelist = map[string]bool{
	"terra10nmmwe8r3g99a9newtqa7a75xfgs2e8z87r2sf": true,
	"terra1gdxfmwcfyrqv8uenllqn7mh290v7dk7x5qnz03": true,
	"terra18hf7422vyyc447uh3wpzm50wzr54welhxlytfg": true,
	"terra1t74f2ahytt9uje3td2lnyv3fkay2jj2akj7ytv": true,
	"terra1qwzdua7928ugklpytdzhua92gnkxp9z4vhelq8": true,
}

// RemoveContractBalances removes contract holding from snapshot, except for the whitelists
func RemoveContractBalances(snapshot SnapshotBalanceAggregateMap, contractMap common.ContractsMap) {
	for contractAddress, _ := range contractMap {
		if _, whitelist := contractWhitelist[contractAddress]; !whitelist {
			delete(snapshot, contractAddress)
		}
	}
}
