package app

var (
	PairVKRUSTAstroport = "terra15s2wgdeqhuc4gfg7sfjyaep5cch38mwtzmwqrx"
	PairVKRUSTTerraswap = "terra1e59utusv5rspqsu8t37h5w887d9rdykljedxw0"

	ValkyrieLPStakingAstroport = "terra1wjc6zd6ue5sqmyucdu8erxj5cdf783tqle6dja"
	ValkyrieLPStakingTerraswap = "terra1ude6ggsvwrhefw2dqjh4j6r7fdmu9nk6nf2z32"
)

// ExportValkyrie iterates over VKR-UST LP then extract UST portion
func ExportValkyrie(app *TerraApp, b blacklist) (map[string]balance, error) {
	b.RegisterAddress(PairVKRUSTAstroport, DenomUST)
	b.RegisterAddress(PairVKRUSTTerraswap, DenomUST)

	// 1. Virtually unstake astroport LP, and add to provided LP

	// 2. Filter UST

	// All dexes are taken into account separately; don't care

	return nil, nil

}
