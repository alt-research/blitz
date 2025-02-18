package fp_instance

import (
	bbntypes "github.com/babylonlabs-io/babylon/types"
	"github.com/btcsuite/btcd/btcec/v2"

	"github.com/babylonlabs-io/finality-provider/finality-provider/proto"
	"github.com/babylonlabs-io/finality-provider/finality-provider/store"
)

func (fp *FinalityProviderInstance) GetStoreFinalityProvider() *store.StoredFinalityProvider {
	return fp.inner.GetStoreFinalityProvider()
}

func (fp *FinalityProviderInstance) GetBtcPkBIP340() *bbntypes.BIP340PubKey {
	return fp.inner.GetBtcPkBIP340()
}

func (fp *FinalityProviderInstance) GetBtcPk() *btcec.PublicKey {
	return fp.inner.GetBtcPk()
}

func (fp *FinalityProviderInstance) GetBtcPkHex() string {
	return fp.inner.GetBtcPkHex()
}

func (fp *FinalityProviderInstance) GetStatus() proto.FinalityProviderStatus {
	return fp.inner.GetStatus()
}

func (fp *FinalityProviderInstance) GetLastVotedHeight() uint64 {
	return fp.inner.GetLastVotedHeight()
}

func (fp *FinalityProviderInstance) GetChainID() []byte {
	return fp.inner.GetChainID()
}

func (fp *FinalityProviderInstance) SetStatus(s proto.FinalityProviderStatus) error {
	return fp.inner.SetStatus(s)
}

func (fp *FinalityProviderInstance) MustSetStatus(s proto.FinalityProviderStatus) {
	fp.inner.MustSetStatus(s)
}

func (fp *FinalityProviderInstance) MustUpdateStateAfterFinalitySigSubmission(height uint64) {
	fp.inner.MustUpdateStateAfterFinalitySigSubmission(height)
}
