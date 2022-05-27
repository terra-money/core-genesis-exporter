package app

import (
	"fmt"

	"github.com/terra-money/core/app/export/alice"
	"github.com/terra-money/core/app/export/anchor"
	"github.com/terra-money/core/app/export/angel"
	"github.com/terra-money/core/app/export/aperture"
	"github.com/terra-money/core/app/export/astroport"
	"github.com/terra-money/core/app/export/edge"
	"github.com/terra-money/core/app/export/generic"
	"github.com/terra-money/core/app/export/ink"
	"github.com/terra-money/core/app/export/kinetic"
	"github.com/terra-money/core/app/export/kujira"
	"github.com/terra-money/core/app/export/lido"
	"github.com/terra-money/core/app/export/loop"
	"github.com/terra-money/core/app/export/native"
	"github.com/terra-money/core/app/export/nebula"
	"github.com/terra-money/core/app/export/prism"
	"github.com/terra-money/core/app/export/pylon"
	"github.com/terra-money/core/app/export/randomearth"
	"github.com/terra-money/core/app/export/stader"
	"github.com/terra-money/core/app/export/starflet"
	"github.com/terra-money/core/app/export/starterra"
	"github.com/terra-money/core/app/export/suberra"
	"github.com/terra-money/core/app/export/terrafloki"
	"github.com/terra-money/core/app/export/terraswap"
	"github.com/terra-money/core/app/export/whitewhale"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/apollo"
	"github.com/terra-money/core/app/export/mars"
	"github.com/terra-money/core/app/export/mirror"
	"github.com/terra-money/core/app/export/spectrum"
	"github.com/terra-money/core/app/export/util"
)

func ExportContracts(app *terra.TerraApp) []types.Balance {
	// var err error

	bl := NewBlacklist()
	logger := app.Logger()
	logger.Info(fmt.Sprintf("Exporting Contracts @ %d", app.LastBlockHeight()))

	// a global holder for all contracts and their contractInfo
	// Export generics
	genericsSnapshot, _, err := generic.ExportGenericContracts(app, bl)
	if err != nil {
		panic(err)
	}

	// Export anchor
	aUST := checkWithSs(util.CachedSBA(anchor.ExportAnchorDeposit, "./anchor.json", app, bl))
	bLunaInCustody := checkWithSs(util.CachedSBA(anchor.ExportbLUNA, "./anchor-bluna.json", app, bl))

	// Export Compounders
	compoundedLps, err := exportCompounders(app)
	if err != nil {
		panic(err)
	}

	check(mirror.AuditCompounders(app, compoundedLps))

	// Export DEXs
	astroportSnapshot := checkWithSs(astroport.ExportAstroportLP(app, bl, compoundedLps))
	terraswapSnapshot := checkWithSs(terraswap.ExportTerraswapLiquidity(app, bl, compoundedLps))
	loopSnapshot := checkWithSs(loop.ExportLoopLP(app, bl))

	// Export Vaults
	suberraSs := checkWithSs(suberra.ExportSuberra(app, bl))
	check(suberra.Audit(app, suberraSs))
	whiteWhaleSs := checkWithSs(whitewhale.ExportWhiteWhaleVaults(app, bl))
	check(whitewhale.Audit(app, whiteWhaleSs))
	kujiraSs := checkWithSs(kujira.ExportKujiraVault(app, bl))
	check(kujira.Audit(app, kujiraSs))
	prismSs := checkWithSs(prism.ExportContract(app, bl))
	check(prism.Audit(app, prismSs))
	prismLoSs := checkWithSs(prism.ExportLimitOrderContract(app, bl))
	check(prism.AuditLOs(app, prismLoSs))
	apertureSs := checkWithSs(util.CachedSBA(aperture.ExportApertureVaultsPreAttack, "./aperture-pre.json", app, bl))
	edgeSs := checkWithSs(edge.ExportContract(app, bl))
	check(edge.Audit(app, edgeSs))
	mirrorSs := checkWithSs(util.CachedSBA(mirror.ExportMirrorCdps, "./mirror-cdp.json", app, bl))
	check(mirror.AuditCdps(app, mirrorSs))
	mirrorLoSs := checkWithSs(mirror.ExportLimitOrderContract(app, bl))
	check(mirror.AuditLOs(app, mirrorLoSs))
	inkSs := checkWithSs(ink.ExportContract(app, bl))
	lunaXSs := checkWithSs(stader.ExportLunaX(app, bl))
	staderPoolSs := checkWithSs(stader.ExportPools(app, bl))
	staderStakeSs := checkWithSs(stader.ExportStakePlus(app, bl))
	staderVaultSs := checkWithSs(stader.ExportVaults(app, bl))
	angelSs := checkWithSs(angel.ExportEndowments(app, bl))
	randomEarthSs := checkWithSs(util.CachedSBA(randomearth.ExportSettlements, "./radomearth.json", app, bl))
	starTerraSs := checkWithSs(starterra.ExportIDO(app, bl))
	check(starterra.Audit(app, starTerraSs))
	marsSs := checkWithSs(mars.ExportContract(app, bl))
	check(mars.Audit(app, marsSs))
	starfletSs := checkWithSs(starflet.ExportArbitrageAUST(app, &bl))
	pylonSs := checkWithSs(pylon.ExportContract(app, bl))
	check(pylon.Audit(app, pylonSs))

	// Export miscellaneous
	flokiSs := checkWithSs(terrafloki.ExportTerraFloki(app, bl))
	flokiRefundsSs := checkWithSs(terrafloki.ExportFlokiRefunds(app, bl))
	nebulaSs := checkWithSs(nebula.ExportNebulaCommunityFund(app, bl))
	aliceSs := checkWithSs(alice.ExportAlice(app, bl))
	kineticSs := checkWithSs(kinetic.ExportKinetic(app, bl))

	snapshot := util.MergeSnapshots(
		genericsSnapshot,
		// DEX
		astroportSnapshot, terraswapSnapshot, loopSnapshot,
		suberraSs, whiteWhaleSs, kujiraSs, prismSs,
		prismLoSs, apertureSs, edgeSs, mirrorSs,
		mirrorLoSs, inkSs, lunaXSs, staderPoolSs,
		staderStakeSs, staderVaultSs, angelSs,
		randomEarthSs, starfletSs, flokiSs,
		flokiRefundsSs, nebulaSs, aliceSs, kineticSs,
		// anchor
		aUST,
		bLunaInCustody,
	)

	// Export Liquid Staking
	check(lido.ExportBSTLunaHolders(app, snapshot, bl))
	check(lido.ExportLidoRewards(app, snapshot, bl))
	check(lido.ResolveLidoLuna(app, snapshot, bl))
	check(prism.ResolveToLuna(app, snapshot, bl))

	bondedLuna := checkWithSs(native.ExportAllBondedLuna(app))
	nativeBalances := checkWithSs(native.ExportAllNativeBalances(app))

	snapshot = util.MergeSnapshots(snapshot, bondedLuna, nativeBalances)
	snapshot.ApplyBlackList(bl)

	finalAudit(app, snapshot, util.Snapshot(util.PreAttack))

	return snapshot.ExportToBalances()
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
	specLps, err := util.CachedMap3(spectrum.ExportSpecVaultLPs, "./spectrum.json", app)
	if err != nil {
		return nil, err
	}
	for k, v := range specLps {
		finalMap[k] = v
	}
	apolloLps, err := util.CachedMap3(apollo.ExportApolloVaultLPs, "./apollo.json", app)
	if err != nil {
		return nil, err
	}
	for k, v := range apolloLps {
		finalMap[k] = v
	}
	marsLps, err := util.CachedMap3(mars.ExportFieldOfMarsLpTokens, "./mars-field.json", app)
	if err != nil {
		return nil, err
	}
	for k, v := range marsLps {
		finalMap[k] = v
	}
	mirrorLps, err := util.CachedMap3(mirror.ExportMirrorLpStakers, "./mirror.json", app)
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

func finalAudit(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap, snapshotType util.Snapshot) error {
	app.Logger().Info("Final audit")
	ctx := util.PrepCtx(app)
	q := util.PrepWasmQueryServer(app)

	// assert no other staking derivatives exist in the snapshot
	util.AssertZeroSupply(snapshot, util.DenomBLUNA)
	util.AssertZeroSupply(snapshot, util.DenomSTLUNA)
	util.AssertZeroSupply(snapshot, util.DenomSTEAK)
	util.AssertZeroSupply(snapshot, util.DenomNLUNA)
	util.AssertZeroSupply(snapshot, util.DenomCLUNA)
	util.AssertZeroSupply(snapshot, util.DenomPLUNA)
	util.AssertZeroSupply(snapshot, util.DenomLUNAX)

	if snapshotType == util.Snapshot(util.PreAttack) {
		// expect to have aUST in the snapshot
		aUstHoldings := snapshot.FilterByDenom(util.DenomAUST)
		util.AssertCw20Supply(ctx, q, util.AUST, aUstHoldings)

		// expect to have LUNA in the snapshot
		lunaHoldings := snapshot.FilterByDenom(util.DenomLUNA)
		util.AssertNativeSupply(ctx, app.BankKeeper, util.DenomLUNA, lunaHoldings)
	} else {
		// expect to have UST in the snapshot
		ustHoldings := snapshot.FilterByDenom(util.DenomUST)
		util.AssertNativeSupply(ctx, app.BankKeeper, util.AUST, ustHoldings)

		// expect to have LUNA in the snapshot
		lunaHoldings := snapshot.FilterByDenom(util.DenomLUNA)
		util.AssertNativeSupply(ctx, app.BankKeeper, util.DenomLUNA, lunaHoldings)
	}
	return nil
}
