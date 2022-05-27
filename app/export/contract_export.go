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
	"github.com/terra-money/core/app/export/nexus"
	"github.com/terra-money/core/app/export/prism"
	"github.com/terra-money/core/app/export/pylon"
	"github.com/terra-money/core/app/export/randomearth"
	"github.com/terra-money/core/app/export/stader"
	"github.com/terra-money/core/app/export/starflet"
	"github.com/terra-money/core/app/export/starterra"
	"github.com/terra-money/core/app/export/steak"
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
	var snapshotType util.Snapshot
	if app.LastBlockHeight() == 7544910 {
		snapshotType = util.Snapshot(util.PreAttack)
	} else {
		snapshotType = util.Snapshot(util.PostAttack)
	}

	bl := NewBlacklist()
	logger := app.Logger()
	logger.Info(fmt.Sprintf("Exporting Contracts @ %d - %s", app.LastBlockHeight(), snapshotType))

	// a global holder for all contracts and their contractInfo
	// Export generics
	genericsSnapshot, contractMap, err := generic.ExportGenericContracts(app, bl)
	if err != nil {
		panic(err)
	}

	// // Export anchor
	aUST := checkWithSs(util.CachedSBA(anchor.ExportAnchorDeposit, "anchor", app, bl))
	bLunaInCustody := checkWithSs(util.CachedSBA(anchor.ExportbLUNA, "anchor-bluna", app, bl))

	// Export Compounders
	compoundedLps, err := exportCompounders(app, snapshotType)
	if err != nil {
		panic(err)
	}

	check(mirror.AuditCompounders(app, compoundedLps))

	// Export DEXs
	astroportSnapshot := checkWithSs(astroport.ExportAstroportLP(app, bl, compoundedLps))
	terraswapSnapshot := checkWithSs(terraswap.ExportTerraswapLiquidity(app, bl, compoundedLps))
	loopSnapshot := checkWithSs(util.CachedSBA(loop.ExportLoopLP, "loop", app, bl))

	// Export Vaults
	suberraSs := checkWithSs(util.CachedSBA(suberra.ExportSuberra, "suberra", app, bl))
	check(suberra.Audit(app, suberraSs))
	whiteWhaleSs := checkWithSs(util.CachedSBA(whitewhale.ExportWhiteWhaleVaults, "whitewhale", app, bl))
	check(whitewhale.Audit(app, whiteWhaleSs))
	kujiraSs := checkWithSs(util.CachedSBA(kujira.ExportKujiraVault, "kujira", app, bl))
	check(kujira.Audit(app, kujiraSs))
	prismSs := checkWithSs(util.CachedSBA(prism.ExportContract, "prism", app, bl))
	check(prism.Audit(app, prismSs))
	prismLoSs := checkWithSs(util.CachedSBA(prism.ExportLimitOrderContract, "prism-limit-order", app, bl))
	check(prism.AuditLOs(app, prismLoSs))
	var apertureSs util.SnapshotBalanceAggregateMap
	if snapshotType == util.Snapshot(util.PreAttack) {
		apertureSs = checkWithSs(util.CachedSBA(aperture.ExportApertureVaultsPreAttack, "aperture-pre", app, bl))
	} else {
		apertureSs = checkWithSs(util.CachedSBA(aperture.ExportApertureVaultsPostAttack, "aperture-post", app, bl))
	}

	edgeSs := checkWithSs(util.CachedSBA(edge.ExportContract, "edge", app, bl))
	check(edge.Audit(app, edgeSs))
	mirrorSs := checkWithSs(util.CachedSBA(mirror.ExportMirrorCdps, "mirror-cdp", app, bl))
	check(mirror.AuditCdps(app, mirrorSs))
	mirrorLoSs := checkWithSs(util.CachedSBA(mirror.ExportLimitOrderContract, "mirror-limit-order", app, bl))
	check(mirror.AuditLOs(app, mirrorLoSs))
	inkSs := checkWithSs(util.CachedSBA(ink.ExportContract, "ink", app, bl))
	lunaXSs := checkWithSs(util.CachedSBA(stader.ExportLunaX, "stader", app, bl))
	staderPoolSs := checkWithSs(util.CachedSBA(stader.ExportPools, "stader-pools", app, bl))
	staderStakeSs := checkWithSs(util.CachedSBA(stader.ExportStakePlus, "stader-stake-plus", app, bl))
	staderVaultSs := checkWithSs(util.CachedSBA(stader.ExportVaults, "stader-vaults", app, bl))
	angelSs := checkWithSs(util.CachedSBA(angel.ExportEndowments, "angel", app, bl))
	randomEarthSs := checkWithSs(util.CachedSBA(randomearth.ExportSettlements, "radomearth", app, bl))
	starTerraSs := checkWithSs(util.CachedSBA(starterra.ExportIDO, "starterra", app, bl))
	check(starterra.Audit(app, starTerraSs))
	starfletSs := checkWithSs(util.CachedSBA(starflet.ExportArbitrageAUST, "starflet", app, bl))
	pylonSs := checkWithSs(util.CachedSBA(pylon.ExportContract, "pylon", app, bl))
	marsSs := make(util.SnapshotBalanceAggregateMap)
	if snapshotType == util.Snapshot(util.PreAttack) {
		marsSs = checkWithSs(util.CachedSBA(mars.ExportContract, "mars", app, bl))
		check(mars.Audit(app, marsSs))
	}

	// Export miscellaneous
	flokiSs := checkWithSs(util.CachedSBA(terrafloki.ExportTerraFloki, "floki", app, bl))
	flokiRefundsSs := checkWithSs(util.CachedSBA(terrafloki.ExportFlokiRefunds, "floki-refunds", app, bl))
	nebulaSs := checkWithSs(util.CachedSBA(nebula.ExportNebulaCommunityFund, "nebula", app, bl))
	aliceSs := checkWithSs(util.CachedSBA(alice.ExportAlice, "alice", app, bl))
	kineticSs := checkWithSs(util.CachedSBA(kinetic.ExportKinetic, "kinetic", app, bl))
	steakSs := checkWithSs(util.CachedSBA(steak.ExportSteak, "steak", app, bl))
	astroportLockDropSs := checkWithSs(util.CachedSBA(astroport.ExportAstroportLockdrop, "astroport-lockdrop", app, bl))
	nexusSs, err := nexus.ExportNexus(app, astroportSnapshot, bl)
	check(err)

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
		steakSs, astroportLockDropSs, nexusSs, marsSs,
		pylonSs,
		// anchor
		aUST,
		bLunaInCustody,
	)

	// Export Liquid Staking
	check(nexus.ResolveToBLuna(app, snapshot, bl))
	check(lido.ExportBSTLunaHolders(app, snapshot, bl))
	check(lido.ExportLidoRewards(app, snapshot, bl))
	check(lido.ResolveLidoLuna(app, snapshot, bl))
	check(prism.ResolveToLuna(app, snapshot, bl))

	bondedLuna := checkWithSs(native.ExportAllBondedLuna(app, bl))
	nativeBalances := checkWithSs(native.ExportAllNativeBalances(app))

	snapshot = util.MergeSnapshots(snapshot, bondedLuna, nativeBalances)
	snapshot.ApplyBlackList(bl)

	if snapshotType == util.Snapshot(util.PostAttack) {
		for _, sbs := range snapshot {
			for _, b := range sbs {
				if b.Denom == util.DenomAUST {
					b.Denom = util.DenomUST
				}
			}
		}
	}

	// remove all contract holdings from snapshot, minus some whitelisted ones
	util.RemoveContractBalances(snapshot, contractMap)

	finalAudit(app, snapshot, snapshotType)

	return snapshot.ExportToBalances()
}

func NewBlacklist() util.Blacklist {
	return util.Blacklist{
		util.DenomUST:  []string{},
		util.DenomLUNA: []string{},
		util.DenomAUST: []string{},
	}
}

func exportCompounders(app *terra.TerraApp, snaphotType util.Snapshot) (map[string]map[string]map[string]sdk.Int, error) {
	finalMap := make(map[string]map[string]map[string]sdk.Int)
	specLps, err := util.CachedMap3(spectrum.ExportSpecVaultLPs, "spectrum", app)
	if err != nil {
		return nil, err
	}
	for k, v := range specLps {
		finalMap[k] = v
	}
	apolloLps, err := util.CachedMap3(apollo.ExportApolloVaultLPs, "apollo", app)
	if err != nil {
		return nil, err
	}
	for k, v := range apolloLps {
		finalMap[k] = v
	}
	if snaphotType == util.Snapshot(util.PreAttack) {
		marsLps, err := util.CachedMap3(mars.ExportFieldOfMarsLpTokens, "mars-field", app)
		if err != nil {
			return nil, err
		}
		for k, v := range marsLps {
			finalMap[k] = v
		}
	}
	mirrorLps, err := util.CachedMap3(mirror.ExportMirrorLpStakers, "mirror", app)
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
	// util.AssertZeroSupply(snapshot, util.AUST) // prevent accidental address as denom
	// util.AssertZeroSupply(snapshot, util.DenomBLUNA)
	// util.AssertZeroSupply(snapshot, util.DenomSTLUNA)
	// util.AssertZeroSupply(snapshot, util.DenomSTEAK)
	// util.AssertZeroSupply(snapshot, util.DenomNLUNA)
	// util.AssertZeroSupply(snapshot, util.DenomCLUNA)
	// util.AssertZeroSupply(snapshot, util.DenomPLUNA)
	// util.AssertZeroSupply(snapshot, util.DenomLUNAX)

	if snapshotType == util.Snapshot(util.PreAttack) {
		// expect to have aUST in the snapshot
		aUstHoldings := snapshot.FilterByDenom(util.DenomAUST)
		err := util.AssertCw20Supply(ctx, q, util.AUST, aUstHoldings)
		if err != nil {
			app.Logger().Info(err.Error())
		}

		// expect to have LUNA in the snapshot
		lunaHoldings := snapshot.FilterByDenom(util.DenomLUNA)
		err = util.AssertNativeSupply(ctx, app.BankKeeper, util.DenomLUNA, lunaHoldings)
		if err != nil {
			app.Logger().Info(err.Error())
		}
	} else {
		// expect to have UST in the snapshot
		ustHoldings := snapshot.FilterByDenom(util.DenomUST)
		err := util.AssertNativeSupply(ctx, app.BankKeeper, util.DenomUST, ustHoldings)
		if err != nil {
			app.Logger().Info(err.Error())
		}

		// expect to have LUNA in the snapshot
		lunaHoldings := snapshot.FilterByDenom(util.DenomLUNA)
		err = util.AssertNativeSupply(ctx, app.BankKeeper, util.DenomLUNA, lunaHoldings)
		if err != nil {
			app.Logger().Info(err.Error())
		}
	}
	return nil
}
