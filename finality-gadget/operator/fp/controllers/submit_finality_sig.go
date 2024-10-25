package controllers

import (
	"context"
	"encoding/json"

	sdkErr "cosmossdk.io/errors"
	wasmdtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	bbntypes "github.com/babylonlabs-io/babylon/types"
	"github.com/babylonlabs-io/finality-provider/types"
	fptypes "github.com/babylonlabs-io/finality-provider/types"
	"github.com/btcsuite/btcd/btcec/v2"
	cmtcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/relayer/v2/relayer/provider"
)

func fromCosmosEventsToBytes(events []provider.RelayerEvent) []byte {
	bytes, err := json.Marshal(events)
	if err != nil {
		return nil
	}
	return bytes
}

type Proof struct {
	Total    int64    `json:"total"`
	Index    int64    `json:"index"`
	LeafHash []byte   `json:"leaf_hash"`
	Aunts    [][]byte `json:"aunts"`
}

type SubmitFinalitySignature struct {
	FpPubkeyHex string `json:"fp_pubkey_hex"`
	Height      uint64 `json:"height"`
	PubRand     []byte `json:"pub_rand"`
	Proof       Proof  `json:"proof"` // nested struct
	BlockHash   []byte `json:"block_hash"`
	Signature   []byte `json:"signature"`
}

type ExecMsg struct {
	SubmitFinalitySignature *SubmitFinalitySignature `json:"submit_finality_signature,omitempty"`
}

// SubmitFinalitySig submits the finality signature to the consumer chain
func (wc *OrbitConsumerController) SubmitFinalitySig(
	fpPk *btcec.PublicKey,
	block *types.BlockInfo,
	pubRand *btcec.FieldVal,
	proof []byte,
	sig *btcec.ModNScalar) (*types.TxResponse, error) {
	wc.logger.Sugar().Debugf("SubmitFinalitySig %v", block)

	cmtProof := cmtcrypto.Proof{}
	if err := cmtProof.Unmarshal(proof); err != nil {
		return nil, err
	}

	aunts := cmtProof.Aunts
	if aunts == nil {
		aunts = [][]byte{}
	}

	proofJSON := Proof{
		Total:    cmtProof.Total,
		Index:    cmtProof.Index,
		LeafHash: cmtProof.LeafHash,
		Aunts:    aunts,
	}

	msg := ExecMsg{
		SubmitFinalitySignature: &SubmitFinalitySignature{
			FpPubkeyHex: bbntypes.NewBIP340PubKeyFromBTCPK(fpPk).MarshalHex(),
			Height:      block.Height,
			PubRand:     bbntypes.NewSchnorrPubRandFromFieldVal(pubRand).MustMarshal(),
			Proof:       proofJSON,
			BlockHash:   block.Hash,
			Signature:   bbntypes.NewSchnorrEOTSSigFromModNScalar(sig).MustMarshal(),
		},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	wc.logger.Sugar().Debugf("submitFinalitySignature %v", string(payload))

	execMsg := &wasmtypes.MsgExecuteContract{
		Sender:   wc.inner.CwClient.MustGetAddr(),
		Contract: wc.cfg.OPFinalityGadgetAddress,
		Msg:      payload,
	}

	res, err := wc.inner.ReliablySendMsg(execMsg, nil, nil)
	if err != nil {
		return nil, err
	}

	tx := &fptypes.TxResponse{TxHash: res.TxHash, Events: fromCosmosEventsToBytes(res.Events)}

	if err != nil {
		wc.logger.Sugar().Errorf("SubmitFinalitySig %v failed: %v", block, err)
	}
	return tx, err
}

// SubmitBatchFinalitySigs submits a batch of finality signatures to Babylon
func (wc *OrbitConsumerController) submitBatchFinalitySigs(
	fpPk *btcec.PublicKey,
	blocks []*fptypes.BlockInfo,
	pubRandList []*btcec.FieldVal,
	proofList [][]byte,
	sigs []*btcec.ModNScalar,
) (*fptypes.TxResponse, error) {
	msgs := make([]sdk.Msg, 0, len(blocks))
	for i, b := range blocks {
		cmtProof := cmtcrypto.Proof{}
		if err := cmtProof.Unmarshal(proofList[i]); err != nil {
			return nil, err
		}

		aunts := cmtProof.Aunts
		if aunts == nil {
			aunts = [][]byte{}
		}

		proofJSON := Proof{
			Total:    cmtProof.Total,
			Index:    cmtProof.Index,
			LeafHash: cmtProof.LeafHash,
			Aunts:    aunts,
		}

		msg := ExecMsg{
			SubmitFinalitySignature: &SubmitFinalitySignature{
				FpPubkeyHex: bbntypes.NewBIP340PubKeyFromBTCPK(fpPk).MarshalHex(),
				Height:      b.Height,
				PubRand:     bbntypes.NewSchnorrPubRandFromFieldVal(pubRandList[i]).MustMarshal(),
				Proof:       proofJSON,
				BlockHash:   b.Hash,
				Signature:   bbntypes.NewSchnorrEOTSSigFromModNScalar(sigs[i]).MustMarshal(),
			},
		}

		msgBytes, err := json.Marshal(msg)
		if err != nil {
			return nil, err
		}

		execMsg := &wasmdtypes.MsgExecuteContract{
			Sender:   wc.inner.CwClient.MustGetAddr(),
			Contract: sdk.MustAccAddressFromBech32(wc.cfg.OPFinalityGadgetAddress).String(),
			Msg:      msgBytes,
		}
		msgs = append(msgs, execMsg)
	}

	res, err := wc.reliablySendMsgs(msgs, nil, nil)
	if err != nil {
		return nil, err
	}

	return &fptypes.TxResponse{TxHash: res.TxHash}, nil
}

func (wc *OrbitConsumerController) reliablySendMsgs(msgs []sdk.Msg, expectedErrs []*sdkErr.Error, unrecoverableErrs []*sdkErr.Error) (*provider.RelayerTxResponse, error) {
	return wc.cwClient.ReliablySendMsgs(
		context.Background(),
		msgs,
		expectedErrs,
		unrecoverableErrs,
	)
}
