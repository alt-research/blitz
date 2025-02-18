package fp_instance

import (
	bbntypes "github.com/babylonlabs-io/babylon/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"

	"github.com/babylonlabs-io/finality-provider/types"
)

func (fp *FinalityProviderInstance) GetPubRandList(startHeight uint64, numPubRand uint32) ([]*btcec.FieldVal, error) {
	return fp.inner.GetPubRandList(startHeight, numPubRand)
}

func (fp *FinalityProviderInstance) SignPubRandCommit(startHeight uint64, numPubRand uint64, commitment []byte) (*schnorr.Signature, error) {
	return fp.inner.SignPubRandCommit(startHeight, numPubRand, commitment)
}

func (fp *FinalityProviderInstance) SignFinalitySig(b *types.BlockInfo) (*bbntypes.SchnorrEOTSSig, error) {
	return fp.SignFinalitySig(b)
}
