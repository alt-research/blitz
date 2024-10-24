package cwclient

import (
	"context"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/ethereum/go-ethereum/common"

	fptypes "github.com/babylonlabs-io/finality-provider/types"
)

type ICosmosWasmContractClient interface {
	QueryListOfVotedFinalityProviders(
		ctx context.Context,
		height uint64,
		hash common.Hash,
	) ([]string, error)

	QueryConfig(ctx context.Context) (contractConfigResponse, error)

	QueryConsumerId(ctx context.Context) (string, error)

	QueryFirstPubRandCommit(ctx context.Context, btcPkHex string) (*PubRandCommit, error)

	QueryLastPubRandCommit(ctx context.Context, btcPkHex string) (*PubRandCommit, error)

	QueryIsEnabled(ctx context.Context) (bool, error)

	// Commit pub rand to wasm contract
	CommitPublicRandomness(ctx context.Context, startHeight uint64, numPubRand uint64, commitment []byte, signature []byte) error

	/// Submit Finality Signature.
	///
	/// This is a message that can be called by a finality provider to submit their finality
	/// signature to the Consumer chain.
	/// The signature is verified by the Consumer chain using the finality provider's public key
	///
	/// This message is equivalent to the `MsgAddFinalitySig` message in the Babylon finality protobuf
	/// defs.
	SubmitFinalitySignature(
		ctx context.Context,
		height uint64,
		pubRand []byte,
		proof crypto.Proof,
		blockHash common.Hash,
		signature []byte) (*fptypes.TxResponse, error)

	SubmitBatchFinalitySignatures(
		ctx context.Context,
		fpPk *btcec.PublicKey,
		blocks []*fptypes.BlockInfo,
		pubRandList []*btcec.FieldVal,
		proofList [][]byte,
		sigs []*btcec.ModNScalar) (*fptypes.TxResponse, error)

	// Commit pub rand to wasm contract
	CommitPublicRandomnessByPK(
		ctx context.Context,
		fpPk *btcec.PublicKey,
		startHeight uint64,
		numPubRand uint64,
		commitment []byte,
		signature []byte) (*fptypes.TxResponse, error)

	/// Submit Finality Signature.
	///
	/// This is a message that can be called by a finality provider to submit their finality
	/// signature to the Consumer chain.
	/// The signature is verified by the Consumer chain using the finality provider's public key
	///
	/// This message is equivalent to the `MsgAddFinalitySig` message in the Babylon finality protobuf
	/// defs.
	SubmitFinalitySignatureByPK(
		ctx context.Context,
		fpPk *btcec.PublicKey,
		height uint64,
		pubRand []byte,
		proof crypto.Proof,
		blockHash common.Hash,
		signature []byte) (*fptypes.TxResponse, error)
}
