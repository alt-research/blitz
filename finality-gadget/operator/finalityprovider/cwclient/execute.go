package cwclient

import (
	"context"
	"encoding/json"

	fptypes "github.com/babylonlabs-io/finality-provider/types"
	"github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"

	wdtypes "github.com/CosmWasm/wasmd/x/wasm/types"
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

	execMsg, err := json.Marshal(struct {
		FpPubkeyHex string `json:"fp_pubkey_hex"`
		StartHeight uint64 `json:"start_height"`
		NumPubRand  uint64 `json:"num_pub_rand"`
		Commitment  string `json:"commitment"`
		Signature   string `json:"signature"`
	}{
		FpPubkeyHex: c.btcPkHex,
		StartHeight: startHeight,
		NumPubRand:  numPubRand,
		Commitment:  hexutil.Encode(commitment),
		Signature:   hexutil.Encode(signature),
	})
	if err != nil {
		return errors.Wrap(err, "CommitPublicRandomness Marshal msg failed")
	}

	executeMsg := &wdtypes.MsgExecuteContract{
		Sender:   c.fpAddr,
		Contract: c.contractAddr,
		Msg:      []byte(execMsg),
	}

	tx, err := c.ReliablySendMsg(context.Background(), executeMsg, nil, nil)
	if err != nil {
		return errors.Wrap(err, "ReliablySendMsg failed")
	}

	if tx != nil {
		c.logger.Info(
			"CommitPublicRandomness ReliablySendMsg resp",
			"Height", tx.Height,
			"TxHash", tx.TxHash,
			"Events", tx.Events,
		)
	}

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

	execMsg, err := json.Marshal(struct {
		FpPubkeyHex string       `json:"fp_pubkey_hex"`
		Height      uint64       `json:"height"`
		PubRand     string       `json:"pub_rand"`
		Proof       crypto.Proof `json:"proof"`
		BlockHash   string       `json:"block_hash"`
		Signature   string       `json:"signature"`
	}{
		FpPubkeyHex: c.btcPkHex,
		Height:      height,
		PubRand:     hexutil.Encode(pubRand),
		Proof:       proof,
		BlockHash:   blockHash.Hex(),
		Signature:   hexutil.Encode(signature),
	})
	if err != nil {
		return nil, errors.Wrap(err, "SubmitFinalitySignature Marshal msg failed")
	}

	executeMsg := &wdtypes.MsgExecuteContract{
		Sender:   c.fpAddr,
		Contract: c.contractAddr,
		Msg:      []byte(execMsg),
	}

	tx, err := c.ReliablySendMsg(context.Background(), executeMsg, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "SubmitFinalitySignature ReliablySendMsg failed")
	}

	if tx != nil {
		c.logger.Info(
			"SubmitFinalitySignature ReliablySendMsg resp",
			"Height", tx.Height,
			"TxHash", tx.TxHash,
			"Events", tx.Events,
		)
	}

	return &fptypes.TxResponse{
		TxHash: tx.TxHash,
		Events: tx.Events,
	}, nil
}
