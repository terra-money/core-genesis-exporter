package nexus

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	terra "github.com/terra-money/core/app"
	"github.com/terra-money/core/app/export/util"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	AddressBLUNAVault           = "terra1cda4adzngjzcn8quvfu2229s8tedl5t306352x"
	AddressCNLUNA               = "terra1u553zk43jd4rwzc53qrdrq4jc2p8rextyq09dj"
	AddressNLUNA                = "terra10f2mt82kjnkxqj2gepgwl637u2w4ue2z5nhz5j"
	AddressCNLUNAAutoCompounder = "terra1au4h305fn4w3zpka2ql59e0t70jnqzu4mj2txx"
	AddressAnchorOverseer       = "terra1tmnqgvg567ypvsvk6rwsga3srp7e3lg6u0elp8"
)

func ExportNexus(app *terra.TerraApp, fromLP util.SnapshotBalanceAggregateMap, bl util.Blacklist) (util.SnapshotBalanceAggregateMap, error) {
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)

	keeper := app.WasmKeeper

	// get all cnLuna holders, unwrap to nLuna
	var cnLunaHolderMap = make(util.BalanceMap)
	if err := util.GetCW20AccountsAndBalances(ctx, keeper, AddressCNLUNA, cnLunaHolderMap); err != nil {
		return nil, fmt.Errorf("failed to fetch cnLUNA holders: %v", err)
	}

	// get nLUNA balance of cnLuna Autocompounder
	nLunaInAutocompounder, err := util.GetCW20Balance(ctx, qs, AddressNLUNA, AddressCNLUNAAutoCompounder)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nLUNA balance in autocompounder: %v", err)
	}

	// get total cnLUNA supply
	var cnLunaSupply struct {
		TotalSupply sdk.Int `json:"total_supply"`
	}
	if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: AddressCNLUNA,
		QueryMsg:        []byte("{\"token_info\":{}}"),
	}, &cnLunaSupply); err != nil {
		return nil, fmt.Errorf("failed to fetch cnLUNA supply")
	}

	// calc nLUNA <> cnLUNA ratio
	ratio := sdk.NewDecFromInt(cnLunaSupply.TotalSupply).QuoInt(nLunaInAutocompounder)

	// iterate over cnLuna holders, convert it to nLUNA
	var nLunaHolderMap = make(util.BalanceMap)
	for userAddr, cnLunaHolding := range cnLunaHolderMap {
		nLunaHolderMap[userAddr] = ratio.MulInt(cnLunaHolding).TruncateInt()
	}

	// iterate over nLuna holders, add it to nLunaHolderMap
	// (bar pairs from dexes)
	var nLunaHolderMapFlat = make(util.BalanceMap)
	if err := util.GetCW20AccountsAndBalances(ctx, keeper, AddressNLUNA, nLunaHolderMapFlat); err != nil {
		return nil, fmt.Errorf("failed to fetch nLUNA holder")
	}

	// merge holder maps + nLUNA holdings from LP
	blacklist := bl.GetAddressesByDenomMap(util.DenomNLUNA)
	mergednLunaHolderMap := util.MergeMaps(nLunaHolderMap, nLunaHolderMapFlat)

	nAssetTobAssetRatio, err := getnAssetTobAssetRatio(ctx, qs)
	if err != nil {
		return nil, err
	}

	// iterate over merged nLUNA holder map, apply nLUNA -> bLUNA ratio
	var finalBalance = make(util.SnapshotBalanceAggregateMap)
	for userAddr, nLunaHolding := range mergednLunaHolderMap {

		// bar blacklisted addresses (pairs, ...)
		if _, exists := blacklist[userAddr]; exists {
			continue
		}

		bLunaAmount := nAssetTobAssetRatio.MulInt(nLunaHolding)

		// there can't be more than 1 holding -- this is fine
		finalBalance[userAddr] = []util.SnapshotBalance{
			{
				Denom:   util.DenomBLUNA,
				Balance: bLunaAmount.TruncateInt(),
			},
		}
	}

	return finalBalance, nil
}

func getnAssetTobAssetRatio(ctx context.Context, qs wasmtypes.QueryServer) (sdk.Dec, error) {
	// nLUNA -> bLUNA ratio
	// get bLUNA held in collateral
	var collaterals struct {
		Collaterals [][2]string `json:"collaterals"`
	}
	if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: AddressAnchorOverseer,
		QueryMsg:        []byte(fmt.Sprintf("{\"collaterals\":{\"borrower\":\"%s\"}}", AddressBLUNAVault)),
	}, &collaterals); err != nil {
		return sdk.Dec{}, fmt.Errorf("failed to fetch Nexus bLUNA vault collateral: %v", err)
	}

	bLUNAProvision, _ := sdk.NewIntFromString(collaterals.Collaterals[0][1])

	// calc nAsset->bAsset ratio
	var nLunaSupply struct {
		TotalSupply sdk.Int `json:"total_supply"`
	}
	if err := util.ContractQuery(ctx, qs, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: AddressNLUNA,
		QueryMsg:        []byte("{\"token_info\":{}}"),
	}, &nLunaSupply); err != nil {
		return sdk.Dec{}, fmt.Errorf("failed to fetch nLUNA total supply: %v", err)
	}
	nAssetTobAssetRatio := sdk.NewDecFromInt(bLUNAProvision).QuoInt(nLunaSupply.TotalSupply)
	return nAssetTobAssetRatio, nil
}

func ResolveToBLuna(app *terra.TerraApp, snapshot util.SnapshotBalanceAggregateMap, bl util.Blacklist) error {
	ctx := util.PrepCtx(app)
	qs := util.PrepWasmQueryServer(app)

	nAssetTobAssetRatio, err := getnAssetTobAssetRatio(ctx, qs)
	if err != nil {
		return err
	}

	for _, sbs := range snapshot {
		for i, sb := range sbs {
			if sb.Denom == util.DenomNLUNA {
				sbs[i] = util.SnapshotBalance{
					Denom:   util.DenomBLUNA,
					Balance: nAssetTobAssetRatio.MulInt(sb.Balance).TruncateInt(),
				}
			}
		}
	}

	return nil
}
