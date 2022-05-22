package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

type NativeMultiSig struct {
	PubKey  string
	Signers []string
}

type Cw3InitMsg struct {
	Voters []struct {
		Address string `json:"addr"`
		Weight  int    `json:"weight"`
	} `json:"voters"`
}

func GenerateNativeMultiSigFromCw3(app *TerraApp, q wasmtypes.QueryServer) (map[string]NativeMultiSig, error) {
	ctx := prepCtx(app)
	cw3ToMultisig := make(map[string]NativeMultiSig)
	totalNumberOfSeenContracts := 0
	app.WasmKeeper.IterateContractInfo(types.UnwrapSDKContext(ctx), func(ci wasmtypes.ContractInfo) bool {
		totalNumberOfSeenContracts += 1
		if totalNumberOfSeenContracts%50 == 0 {
			fmt.Printf("\r%d", totalNumberOfSeenContracts)
		}
		var cw3InitMsg Cw3InitMsg
		err := json.Unmarshal(ci.InitMsg, &cw3InitMsg)
		if err != nil {
			return false
		}
		if len(cw3InitMsg.Voters) > 0 {
			var signers []string
			var pubKeys []cryptotypes.PubKey
			for _, voter := range cw3InitMsg.Voters {
				acc, err := types.AccAddressFromBech32(voter.Address)
				if err != nil {
					panic(err)
				}
				pubKey, err := app.AccountKeeper.GetPubKey(types.UnwrapSDKContext(ctx), acc)
				if err != nil {
					if strings.Contains(err.Error(), "unknown address") {
						fmt.Printf("unknown: %s\n", voter)
						continue
					}
					panic(err)
				}
				if pubKey != nil {
					pubKeys = append(pubKeys, pubKey)
					signers = append(signers, voter.Address)
				}
			}
			if len(pubKeys) == 0 {
				fmt.Printf("missing signer: %s\n", ci.Address)
				return false
			}
			var threshold int
			if len(pubKeys) < 3 {
				threshold = len(pubKeys)
			} else {
				threshold = 3
			}
			// fmt.Printf("%v", pubKeys)
			mSig := multisig.NewLegacyAminoPubKey(threshold, pubKeys)
			mSigAcc, err := types.AccAddressFromHex(mSig.Address().String())
			if err != nil {
				panic(err)
			}
			cw3ToMultisig[ci.Address] = NativeMultiSig{
				PubKey:  mSigAcc.String(),
				Signers: signers,
			}
		}
		return false
	})
	return cw3ToMultisig, nil
}
