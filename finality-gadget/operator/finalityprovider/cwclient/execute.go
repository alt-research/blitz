package cwclient

import (
	"context"

	fptypes "github.com/babylonlabs-io/finality-provider/types"
	"github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/ethereum/go-ethereum/common"
)

func (c *CosmWasmClient) CommitPublicRandomness(
	ctx context.Context,
	startHeight uint64,
	numPubRand uint64,
	commitment []byte,
	signature []byte) error {
	c.logger.Info(
		"CommitPublicRandomness",
		"fpPubkeyHex", c.btcPkHex,
		"startHeight", startHeight,
		"numPubRand", numPubRand,
	)
	return nil
}

func (c *CosmWasmClient) SubmitFinalitySignature(
	ctx context.Context,
	height uint64,
	pubRand []byte,
	proof crypto.Proof,
	blockHash common.Hash,
	signature []byte) (*fptypes.TxResponse, error) {
	c.logger.Info(
		"SubmitFinalitySignature",
		"fpPubkeyHex", c.btcPkHex,
		"height", height,
		"blockHash", blockHash,
	)

	return nil, nil
}
