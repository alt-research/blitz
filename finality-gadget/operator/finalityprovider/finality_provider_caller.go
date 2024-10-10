package finalityprovider

import (
	"context"

	fptypes "github.com/babylonlabs-io/finality-provider/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/core/types"
)

// Commit pub rand to wasm contract
func (fp *FinalityProvider) CommitPublicRandomness(ctx context.Context, tipHeight uint64) error {
	fp.logger.Info(
		"CommitPublicRandomness",
		"tipHeight", tipHeight,
	)

	return nil
}

// Get pub rand list by number
func (fp *FinalityProvider) GetPublicRandomnessList(ctx context.Context, startHeight uint64, numPubRand uint64) ([]*btcec.FieldVal, error) {
	fp.logger.Info(
		"GetPublicRandomnessList",
		"startHeight", startHeight,
		"numPubRand", numPubRand,
	)

	return nil, nil
}

// Submit finality signature to wasm contract
func (fp *FinalityProvider) SubmitFinalitySignature(ctx context.Context, block *types.Block) (*fptypes.TxResponse, error) {
	fp.logger.Info(
		"SubmitFinalitySignature",
		"number", block.NumberU64(),
		"hash", block.Hash(),
	)

	return nil, nil
}

// SubmitBatchFinalitySignatures builds and sends a finality signature over the given block to the consumer chain
// NOTE: the input blocks should be in the ascending order of height
func (fp *FinalityProvider) SubmitBatchFinalitySignatures(ctx context.Context, blocks []*types.Block) (*fptypes.TxResponse, error) {
	fp.logger.Info(
		"SubmitBatchFinalitySignatures",
		"len", len(blocks),
	)

	return nil, nil
}
