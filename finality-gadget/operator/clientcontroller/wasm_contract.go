package clientcontroller

import (
	"cosmossdk.io/math"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"

	btcstakingtypes "github.com/babylonlabs-io/babylon/x/btcstaking/types"
	finalitytypes "github.com/babylonlabs-io/babylon/x/finality/types"
	fpcontroller "github.com/babylonlabs-io/finality-provider/clientcontroller"
	"github.com/babylonlabs-io/finality-provider/types"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
)

var _ fpcontroller.ClientController = &WasmContractController{}

type WasmContractController struct {
	logger logging.Logger
}

// RegisterFinalityProvider registers a finality provider to the consumer chain
// it returns tx hash and error. The address of the finality provider will be
// the signer of the msg.
func (wc *WasmContractController) RegisterFinalityProvider(
	fpPk *btcec.PublicKey,
	pop []byte,
	commission *math.LegacyDec,
	description []byte,
) (*types.TxResponse, error) {
	return nil, nil
}

// CommitPubRandList commits a list of EOTS public randomness the consumer chain
// it returns tx hash and error
func (wc *WasmContractController) CommitPubRandList(fpPk *btcec.PublicKey, startHeight uint64, numPubRand uint64, commitment []byte, sig *schnorr.Signature) (*types.TxResponse, error) {
	return nil, nil
}

// SubmitFinalitySig submits the finality signature to the consumer chain
func (wc *WasmContractController) SubmitFinalitySig(fpPk *btcec.PublicKey, block *types.BlockInfo, pubRand *btcec.FieldVal, proof []byte, sig *btcec.ModNScalar) (*types.TxResponse, error) {
	return nil, nil
}

// SubmitBatchFinalitySigs submits a batch of finality signatures to the consumer chain
func (wc *WasmContractController) SubmitBatchFinalitySigs(
	fpPk *btcec.PublicKey,
	blocks []*types.BlockInfo,
	pubRandList []*btcec.FieldVal,
	proofList [][]byte,
	sigs []*btcec.ModNScalar) (*types.TxResponse, error) {
	return nil, nil
}

// UnjailFinalityProvider sends an unjail transaction to the consumer chain
func (wc *WasmContractController) UnjailFinalityProvider(fpPk *btcec.PublicKey) (*types.TxResponse, error) {
	return nil, nil
}

// QueryFinalityProviderVotingPower queries the voting power of the finality provider at a given height
func (wc *WasmContractController) QueryFinalityProviderVotingPower(fpPk *btcec.PublicKey, blockHeight uint64) (uint64, error) {
	return 0, nil
}

// QueryFinalityProviderSlashedOrJailed queries if the finality provider is slashed or jailed
func (wc *WasmContractController) QueryFinalityProviderSlashedOrJailed(fpPk *btcec.PublicKey) (slashed bool, jailed bool, err error) {
	return true, true, nil
}

// EditFinalityProvider edits description and commission of a finality provider
func (wc *WasmContractController) EditFinalityProvider(fpPk *btcec.PublicKey, commission *math.LegacyDec, description []byte) (*btcstakingtypes.MsgEditFinalityProvider, error) {
	return nil, nil
}

// QueryLatestFinalizedBlocks returns the latest finalized blocks
func (wc *WasmContractController) QueryLatestFinalizedBlocks(count uint64) ([]*types.BlockInfo, error) {
	return nil, nil
}

// QueryLastCommittedPublicRand returns the last committed public randomness
func (wc *WasmContractController) QueryLastCommittedPublicRand(fpPk *btcec.PublicKey, count uint64) (map[uint64]*finalitytypes.PubRandCommitResponse, error) {
	return nil, nil
}

// QueryBlock queries the block at the given height
func (wc *WasmContractController) QueryBlock(height uint64) (*types.BlockInfo, error) {
	return nil, nil
}

// QueryBlocks returns a list of blocks from startHeight to endHeight
func (wc *WasmContractController) QueryBlocks(startHeight, endHeight uint64, limit uint32) ([]*types.BlockInfo, error) {
	return nil, nil
}

// QueryBestBlock queries the tip block of the consumer chain
func (wc *WasmContractController) QueryBestBlock() (*types.BlockInfo, error) {
	return nil, nil
}

// QueryActivatedHeight returns the activated height of the consumer chain
// error will be returned if the consumer chain has not been activated
func (wc *WasmContractController) QueryActivatedHeight() (uint64, error) {
	return 0, nil
}

func (wc *WasmContractController) Close() error {
	return nil
}
