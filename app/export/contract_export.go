package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/aperture"
	"github.com/terra-money/core/app/export/apollo"
	"github.com/terra-money/core/app/export/astroport"
	"github.com/terra-money/core/app/export/edge"
	"github.com/terra-money/core/app/export/ink"
	"github.com/terra-money/core/app/export/kujira"
	"github.com/terra-money/core/app/export/lido"
	"github.com/terra-money/core/app/export/loop"
	"github.com/terra-money/core/app/export/mars"
	"github.com/terra-money/core/app/export/mirror"
	"github.com/terra-money/core/app/export/prism"
	"github.com/terra-money/core/app/export/spectrum"
	"github.com/terra-money/core/app/export/terraswap"
	"github.com/terra-money/core/app/export/util"
	"github.com/terra-money/core/app/export/whitewhale"
)

func ExportContracts(app *terra.TerraApp) {
	var err error

	bl := NewBlacklist()
	// snapshot := make(util.SnapshotBalanceAggregateMap)

	logger := app.Logger()
	logger.Info(fmt.Sprintf("Exporting Contracts @ %d", app.LastBlockHeight()))

	// Export Compounders
	compoundedLps, err := exportCompounders(app)
	if err != nil {
		panic(err)
	}

	// Export DEXs
	astroportSnapshot := checkWithSs(astroport.ExportAstroportLP(app, bl, compoundedLps))
	terraswapSnapshot := checkWithSs(terraswap.ExportTerraswapLiquidity(app, bl, compoundedLps))
	loopSnapshot := checkWithSs(loop.ExportLoopLP(app, bl))

	// Export Vaults
	whiteWhaleSs := checkWithSs(whitewhale.ExportWhiteWhaleVaults(app, &bl))
	kujiraSs := checkWithSs(kujira.ExportKujiraVault(app, &bl))
	prismSs := checkWithSs(prism.ExportLimitOrderContract(app, &bl))
	apertureSs := checkWithSs(aperture.ExportApertureVaults(app, util.Snapshot(util.PreAttack), &bl))
	edgeSs := checkWithSs(edge.ExportContract(app, &bl))
	mirrorSs := checkWithSs(mirror.ExportMirrorCdps(app, bl))
	mirrorLoSs := checkWithSs(mirror.ExportLimitOrderContract(app, &bl))
	inkSs := checkWithSs(ink.ExportContract(app, &bl))

	// Independent snapshot audits and sanity checks
	check(mirror.Audit(app, mirrorSs))

	snapshot := util.MergeSnapshots(
		terraswapSnapshot, loopSnapshot, astroportSnapshot,
		whiteWhaleSs, kujiraSs, prismSs, apertureSs,
		edgeSs, mirrorSs, mirrorLoSs, inkSs,
	)

	// Export Liquid Staking
	check(lido.ExportBSTLunaHolders(app, snapshot, bl))
	check(lido.ExportLidoRewards(app, snapshot, bl))
	check(lido.ResolveLidoLuna(app, snapshot, bl))
	check(prism.ExportContract(app, snapshot, &bl))
	check(prism.ResolveToLuna(app, snapshot, bl))

	// Final audits and sanity checks (optional?)
	check(mirror.FinalAudit(app, mirrorSs, snapshot))
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
	specLps, err := spectrum.ExportSpecVaultLPs(app)
	if err != nil {
		return nil, err
	}
	for k, v := range specLps {
		finalMap[k] = v
	}
	apolloLps, err := apollo.ExportApolloVaultLPs(app)
	if err != nil {
		return nil, err
	}
	for k, v := range apolloLps {
		finalMap[k] = v
	}
	marsLps, err := mars.ExportFieldOfMarsLpTokens(app)
	if err != nil {
		return nil, err
	}
	for k, v := range marsLps {
		finalMap[k] = v
	}
	mirrorLps, err := mirror.ExportMirrorLpStakers(app)
	if err != nil {
		return nil, err
	}
	for k, v := range mirrorLps {
		finalMap[k] = v
	}
	return finalMap, nil
}

func checkWithSs(snapshot util.SnapshotBalanceAggregateMap, err error) util.SnapshotBalanceAggregateMap {
	if err != nil {
		panic(err)
	}
	return snapshot
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
