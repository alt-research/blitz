package finalityprovider

import (
	"context"

	fptypes "github.com/babylonlabs-io/finality-provider/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

var _ IFinalityProvider = &L2BlockCommitter{}

// Commit pub rand to wasm contract
func (fp *L2BlockCommitter) CommitPublicRandomness(ctx context.Context, tipHeight uint64) error {
	fp.logger.Info(
		"CommitPublicRandomness",
		"tipHeight", tipHeight,
	)

	err := fp.cwClient.CommitPublicRandomness(ctx, tipHeight, fp.cfg.NumPubRand, []byte{0x1}, []byte{0x1})
	if err != nil {
		return errors.Wrapf(err, "CommitPublicRandomness failed by client with tip height %d", tipHeight)
	}

	return nil
}

// Get pub rand list by number
func (fp *L2BlockCommitter) GetPublicRandomnessList(ctx context.Context, startHeight uint64, numPubRand uint64) ([]*btcec.FieldVal, error) {
	fp.logger.Info(
		"GetPublicRandomnessList",
		"startHeight", startHeight,
		"numPubRand", numPubRand,
	)

	return nil, nil
}

// Submit finality signature to wasm contract
func (fp *L2BlockCommitter) SubmitFinalitySignature(ctx context.Context, block *types.Block) (*fptypes.TxResponse, error) {
	fp.logger.Info(
		"SubmitFinalitySignature",
		"number", block.NumberU64(),
		"hash", block.Hash(),
	)

	return nil, nil
}
