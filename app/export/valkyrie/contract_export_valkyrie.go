package app

import (
	terra "github.com/terra-money/core/app"
	util "github.com/terra-money/core/app/export/util"
)

var (
	PairVKRUSTAstroport = "terra15s2wgdeqhuc4gfg7sfjyaep5cch38mwtzmwqrx"
	PairVKRUSTTerraswap = "terra1e59utusv5rspqsu8t37h5w887d9rdykljedxw0"

	ValkyrieLPStakingAstroport = "terra1wjc6zd6ue5sqmyucdu8erxj5cdf783tqle6dja"
	ValkyrieLPStakingTerraswap = "terra1ude6ggsvwrhefw2dqjh4j6r7fdmu9nk6nf2z32"
)

// ExportValkyrie iterates over VKR-UST LP then extract UST portion
func ExportValkyrie(app *terra.TerraApp, b util.Blacklist) (map[string]util.SnapshotBalance, error) {
	b.RegisterAddress(PairVKRUSTAstroport, util.DenomUST)
	b.RegisterAddress(PairVKRUSTTerraswap, util.DenomUST)

	// 1. Virtually unstake astroport LP, and add to provided LP

	// 2. Filter UST

	// All dexes are taken into account separately; don't care

	return nil, nil

}
