package finalityprovider

import (
	"context"

	fptypes "github.com/babylonlabs-io/finality-provider/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/core/types"
)

type IFinalityProvider interface {
	// Commit pub rand to wasm contract
	CommitPublicRandomness(ctx context.Context, tipHeight uint64) error
	// Get pub rand list by number
	GetPublicRandomnessList(ctx context.Context, startHeight uint64, numPubRand uint64) ([]*btcec.FieldVal, error)
	// Submit finality signature to wasm contract
	SubmitFinalitySignature(ctx context.Context, block *types.Block) (*fptypes.TxResponse, error)
	// SubmitBatchFinalitySignatures builds and sends a finality signature over the given block to the consumer chain
	// NOTE: the input blocks should be in the ascending order of height
	SubmitBatchFinalitySignatures(ctx context.Context, blocks []*types.Block) (*fptypes.TxResponse, error)
}
