package cwclient

import (
	"context"
	"encoding/json"

	fptypes "github.com/babylonlabs-io/finality-provider/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/cometbft/cometbft/proto/tendermint/crypto"
	cmtcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"

	wdtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

type CommitPublicRandomness struct {
	FpPubkeyHex string `json:"fp_pubkey_hex"`
	StartHeight uint64 `json:"start_height"`
	NumPubRand  uint64 `json:"num_pub_rand"`
	Commitment  string `json:"commitment"`
	Signature   string `json:"signature"`
}

func (c *CosmWasmClient) CommitPublicRandomness(
	ctx context.Context,
	startHeight uint64,
	numPubRand uint64,
	commitment []byte,
	signature []byte) error {
	_, err := c.CommitPublicRandomnessByPK(ctx, c.btcPk, startHeight, numPubRand, commitment, signature)
	return err
}

func (c *CosmWasmClient) CommitPublicRandomnessByPK(
	ctx context.Context,
	fpPk *btcec.PublicKey,
	startHeight uint64,
	numPubRand uint64,
	commitment []byte,
	signature []byte) (*fptypes.TxResponse, error) {
	btcPkHex := BtcPkToHex(fpPk)
	c.logger.Info(
		"CommitPublicRandomnessByPK",
		"fpPubkeyHex", btcPkHex,
		"startHeight", startHeight,
		"numPubRand", numPubRand,
	)

	msg := make(map[string]json.RawMessage)
	execMsg, err := json.Marshal(
		CommitPublicRandomness{
			FpPubkeyHex: btcPkHex,
			StartHeight: startHeight,
			NumPubRand:  numPubRand,
			Commitment:  hexutil.Encode(commitment),
			Signature:   hexutil.Encode(signature),
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "CommitPublicRandomness Marshal msg failed")
	}
	msg["commit_public_randomness"] = execMsg

	msgBytes, err := json.Marshal(
		msg,
	)
	if err != nil {
		return nil, errors.Wrap(err, "CommitPublicRandomness msgBytes Marshal msg failed")
	}

	executeMsg := &wdtypes.MsgExecuteContract{
		Sender:   c.fpAddr,
		Contract: c.contractAddr,
		Msg:      []byte(msgBytes),
	}

	c.logger.Debug("CommitPublicRandomness msgBytes", "json", string(msgBytes))

	tx, err := c.ReliablySendMsg(ctx, executeMsg, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "ReliablySendMsg failed")
	}

	if tx != nil {
		c.logger.Info(
			"CommitPublicRandomness ReliablySendMsg resp",
			"Height", tx.Height,
			"TxHash", tx.TxHash,
			"Events", tx.Events,
		)
	}

	return &fptypes.TxResponse{
		TxHash: tx.TxHash,
		//Events: tx.Events,
	}, nil
}

func (c *CosmWasmClient) SubmitFinalitySignature(
	ctx context.Context,
	height uint64,
	pubRand []byte,
	proof crypto.Proof,
	blockHash common.Hash,
	signature []byte) (*fptypes.TxResponse, error) {
	return c.SubmitFinalitySignatureByPK(ctx, c.btcPk, height, pubRand, proof, blockHash, signature)
}

func (c *CosmWasmClient) SubmitFinalitySignatureByPK(
	ctx context.Context,
	fpPk *btcec.PublicKey,
	height uint64,
	pubRand []byte,
	proof crypto.Proof,
	blockHash common.Hash,
	signature []byte) (*fptypes.TxResponse, error) {
	btcPkHex := BtcPkToHex(fpPk)
	c.logger.Info(
		"SubmitFinalitySignatureByPK",
		"fpPubkeyHex", btcPkHex,
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
		FpPubkeyHex: btcPkHex,
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

	tx, err := c.ReliablySendMsg(ctx, executeMsg, nil, nil)
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
		//Events: tx.Events,
	}, nil
}

func (c *CosmWasmClient) SubmitBatchFinalitySignatures(
	ctx context.Context,
	fpPk *btcec.PublicKey,
	blocks []*fptypes.BlockInfo,
	pubRandList []*btcec.FieldVal,
	proofList [][]byte,
	sigs []*btcec.ModNScalar) (*fptypes.TxResponse, error) {
	btcPkHex := BtcPkToHex(fpPk)
	c.logger.Info(
		"SubmitBatchFinalitySignatures",
		"fpPubkeyHex", btcPkHex,
	)

	msgs := make([]sdk.Msg, 0, len(blocks))

	for i, b := range blocks {
		pubRandBytes := *pubRandList[i].Bytes()
		cmtProof := cmtcrypto.Proof{}
		if err := cmtProof.Unmarshal(proofList[i]); err != nil {
			return nil, err
		}
		sigBytes := sigs[i].Bytes()

		execMsg, err := json.Marshal(struct {
			FpPubkeyHex string       `json:"fp_pubkey_hex"`
			Height      uint64       `json:"height"`
			PubRand     string       `json:"pub_rand"`
			Proof       crypto.Proof `json:"proof"`
			BlockHash   string       `json:"block_hash"`
			Signature   string       `json:"signature"`
		}{
			FpPubkeyHex: btcPkHex,
			Height:      b.Height,
			PubRand:     hexutil.Encode(pubRandBytes[:]),
			Proof:       cmtProof,
			BlockHash:   common.BytesToHash(b.Hash).Hex(),
			Signature:   hexutil.Encode(sigBytes[:]),
		})
		if err != nil {
			return nil, errors.Wrap(err, "SubmitFinalitySignature Marshal msg failed")
		}

		executeMsg := &wdtypes.MsgExecuteContract{
			Sender:   c.fpAddr,
			Contract: c.contractAddr,
			Msg:      []byte(execMsg),
		}

		msgs = append(msgs, executeMsg)
	}

	tx, err := c.ReliablySendMsgs(ctx, msgs, nil, nil)
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
		//Events: tx.Events,
	}, nil
}
