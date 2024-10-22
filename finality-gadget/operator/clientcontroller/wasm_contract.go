package clientcontroller

import (
	"context"
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	bbnclient "github.com/babylonlabs-io/babylon/client/client"
	bbncfg "github.com/babylonlabs-io/babylon/client/config"
	bbntypes "github.com/babylonlabs-io/babylon/types"
	btcstakingtypes "github.com/babylonlabs-io/babylon/x/btcstaking/types"
	finalitytypes "github.com/babylonlabs-io/babylon/x/finality/types"
	fpcontroller "github.com/babylonlabs-io/finality-provider/clientcontroller"
	"github.com/babylonlabs-io/finality-provider/types"
	cmtcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"

	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	"github.com/alt-research/blitz/finality-gadget/operator/finalityprovider"
	"github.com/alt-research/blitz/finality-gadget/operator/finalityprovider/cwclient"
)

var _ fpcontroller.ClientController = &WasmContractController{}

type WasmContractController struct {
	logger logging.Logger

	ctx context.Context

	bbnClient *bbnclient.Client
	l2Client  *l2eth.L2EthClient
	cwClient  cwclient.ICosmosWasmContractClient

	// The activated_height
	activatedHeight uint64
	consumerId      string
}

func NewWasmContractControllerByConfig(
	ctx context.Context,
	logger logging.Logger,
	zapLogger *zap.Logger,
	cfg *configs.OperatorConfig,
) (*WasmContractController, error) {
	// Create babylon client
	bbnConfig := bbncfg.DefaultBabylonConfig()
	bbnFgCfg := cfg.Babylon.FinalityGadget()
	bbnConfig.RPCAddr = bbnFgCfg.BBNRPCAddress
	bbnConfig.ChainID = bbnFgCfg.BBNChainID
	babylonClient, err := bbnclient.New(
		&bbnConfig,
		zapLogger,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create babylon client")
	}

	l2Client, err := l2eth.NewL2EthClient(ctx, &cfg.Layer2)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create l2 eth client")
	}

	btcPk, err := cfg.FinalityProvider.GetBtcPk()
	if err != nil {
		return nil, errors.Wrap(err, "get btc pk failed")
	}

	cp, err := finalityprovider.NewProvider(ctx, &cfg.FinalityProvider, zapLogger)
	if err != nil {
		return nil, errors.Wrap(err, "new provider failed")
	}

	key, err := cp.GetKeyAddressForKey(cfg.FinalityProvider.Cosmwasm.Key)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"failed to get key address for %v",
			cfg.FinalityProvider.Cosmwasm.Key)
	}

	logger.Debug("key address", "name", cfg.FinalityProvider.Cosmwasm.Key, "address", key)

	cwClient := cwclient.NewCosmWasmClient(
		logger.With("module", "cosmWasmClient"),
		babylonClient.QueryClient.RPCClient,
		btcPk,
		cfg.FinalityProvider.BtcPk,
		cfg.FinalityProvider.FgContractAddress,
		cfg.FinalityProvider.FpAddr,
		cp)

	return &WasmContractController{
		logger:          logger.With("module", "WasmContractController"),
		ctx:             ctx,
		bbnClient:       babylonClient,
		l2Client:        l2Client,
		cwClient:        cwClient,
		activatedHeight: cfg.Layer2.ActivatedHeight,
		consumerId:      cfg.FinalityProvider.ConsumerId,
	}, nil

}

func (wc *WasmContractController) Ctx() context.Context {
	return wc.ctx
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
	// TODO: not support currently
	return nil, errors.Errorf("Not yet supported RegisterFinalityProvider")
}

// CommitPubRandList commits a list of EOTS public randomness the consumer chain
// it returns tx hash and error
func (wc *WasmContractController) CommitPubRandList(
	fpPk *btcec.PublicKey,
	startHeight uint64,
	numPubRand uint64,
	commitment []byte,
	sig *schnorr.Signature) (*types.TxResponse, error) {
	wc.logger.Info(
		"CommitPublicRandomnessByPK",
		"startHeight", startHeight,
		"numPubRand", numPubRand,
	)

	tx, err := wc.cwClient.CommitPublicRandomnessByPK(
		wc.Ctx(),
		fpPk,
		startHeight,
		numPubRand,
		commitment,
		sig.Serialize())
	if err != nil {
		return nil, errors.Wrapf(err, "CommitPubRandList failed by client with tip height %d", startHeight)
	}

	return tx, err
}

// SubmitFinalitySig submits the finality signature to the consumer chain
func (wc *WasmContractController) SubmitFinalitySig(
	fpPk *btcec.PublicKey,
	block *types.BlockInfo,
	pubRand *btcec.FieldVal,
	proof []byte,
	sig *btcec.ModNScalar) (*types.TxResponse, error) {
	wc.logger.Info(
		"SubmitFinalitySig",
		"height", block.Height,
	)

	pubRandBytes := *pubRand.Bytes()
	cmtProof := cmtcrypto.Proof{}
	if err := cmtProof.Unmarshal(proof); err != nil {
		return nil, err
	}
	sigBytes := sig.Bytes()

	tx, err := wc.cwClient.SubmitFinalitySignatureByPK(
		wc.Ctx(),
		fpPk,
		block.Height,
		pubRandBytes[:],
		cmtProof,
		common.BytesToHash(block.Hash),
		sigBytes[:])
	if err != nil {
		return nil, errors.Wrapf(err, "CommitPubRandList failed by client with tip height %d", block.Height)
	}

	return tx, err
}

// SubmitBatchFinalitySigs submits a batch of finality signatures to the consumer chain
func (wc *WasmContractController) SubmitBatchFinalitySigs(
	fpPk *btcec.PublicKey,
	blocks []*types.BlockInfo,
	pubRandList []*btcec.FieldVal,
	proofList [][]byte,
	sigs []*btcec.ModNScalar) (*types.TxResponse, error) {
	blocksLen := len(blocks)
	if blocksLen != len(pubRandList) || blocksLen != len(proofList) || blocksLen != len(sigs) {
		return nil, errors.Errorf(
			"SubmitBatchFinalitySigs failed by len no eq for %v,%v,%v,%v",
			blocksLen, len(pubRandList), len(proofList), len(sigs),
		)
	}

	tx, err := wc.cwClient.SubmitBatchFinalitySignatures(
		wc.Ctx(),
		fpPk,
		blocks,
		pubRandList,
		proofList,
		sigs)
	if err != nil {
		return nil, errors.Wrapf(err, "SubmitBatchFinalitySigs failed by client")
	}

	return tx, err
}

// UnjailFinalityProvider sends an unjail transaction to the consumer chain
func (wc *WasmContractController) UnjailFinalityProvider(fpPk *btcec.PublicKey) (*types.TxResponse, error) {
	return nil, errors.Errorf("Not yet supported UnjailFinalityProvider")
}

// QueryFinalityProviderVotingPower queries the voting power of the finality provider at a given height
func (wc *WasmContractController) QueryFinalityProviderVotingPower(fpPk *btcec.PublicKey, blockHeight uint64) (uint64, error) {
	res, err := wc.bbnClient.QueryClient.FinalityProviderPowerAtHeight(
		bbntypes.NewBIP340PubKeyFromBTCPK(fpPk).MarshalHex(),
		blockHeight,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to query Finality Voting Power at Height %d: %w", blockHeight, err)
	}

	return res.VotingPower, nil
}

// QueryFinalityProviderSlashedOrJailed queries if the finality provider is slashed or jailed
func (wc *WasmContractController) QueryFinalityProviderSlashedOrJailed(fpPk *btcec.PublicKey) (slashed bool, jailed bool, err error) {
	wc.logger.Warn("QueryFinalityProviderSlashedOrJailed just return false currently")
	return false, false, nil
}

// EditFinalityProvider edits description and commission of a finality provider
func (wc *WasmContractController) EditFinalityProvider(
	fpPk *btcec.PublicKey,
	commission *math.LegacyDec,
	description []byte) (*btcstakingtypes.MsgEditFinalityProvider, error) {
	return nil, errors.Errorf("Not yet supported EditFinalityProvider")
}

// QueryLatestFinalizedBlocks returns the latest finalized blocks
func (wc *WasmContractController) QueryLatestFinalizedBlocks(count uint64) ([]*types.BlockInfo, error) {
	if count == 0 {
		wc.logger.Warn("QueryLatestFinalizedBlocks should no zero")
		return nil, nil
	}

	finalized, err := wc.l2Client.BlockReceipts(wc.Ctx(), rpc.BlockNumberOrHashWithNumber(rpc.FinalizedBlockNumber))
	if err != nil {
		return nil, errors.Wrap(err, "failed to QueryLatestFinalizedBlocks by get finalized BlockReceipts")
	}

	if len(finalized) == 0 {
		wc.logger.Warn("QueryLatestFinalizedBlocks no finalized blocks from layer2")
		return nil, nil
	}

	finalizedNumber := finalized[0].BlockNumber.Uint64()

	if finalizedNumber == 0 {
		wc.logger.Warn("QueryLatestFinalizedBlocks finalized block number is zero, return nothing")
		return nil, nil
	}

	if len(finalized) != 1 {
		wc.logger.Warn("QueryLatestFinalizedBlocks the finalized block number not 1",
			"from", finalizedNumber,
			"to", finalized[len(finalized)-1].BlockNumber.Uint64())
	}

	fromNumber := uint64(0)
	if finalizedNumber < count {
		wc.logger.Warn(
			"QueryLatestFinalizedBlocks currently finalized block number less then count",
			"finalized", finalizedNumber, "count", count)
	} else {
		fromNumber = finalizedNumber - count
	}

	res := make([]*types.BlockInfo, 0, count)

	for i := fromNumber; i < finalizedNumber-1; i++ {
		header, err := wc.l2Client.HeaderByNumber(wc.Ctx(), big.NewInt(int64(i)))
		if err != nil {
			return nil, errors.Wrapf(err, "HeaderByNumber failed by %v", i)
		}

		hash := header.Hash()

		res = append(res, &types.BlockInfo{
			Height:    i,
			Hash:      hash[:],
			Finalized: true,
		})
	}

	res = append(res, &types.BlockInfo{
		Height:    finalizedNumber,
		Hash:      finalized[0].BlockHash[:],
		Finalized: true,
	})

	return nil, nil
}

// QueryLastCommittedPublicRand returns the last committed public randomness
func (wc *WasmContractController) QueryLastCommittedPublicRand(
	fpPk *btcec.PublicKey,
	count uint64) (map[uint64]*finalitytypes.PubRandCommitResponse, error) {
	if count != 1 {
		// TODO: now all call is use params 1
		wc.logger.Errorf("QueryLastCommittedPublicRand count not 1 but got %d", count)
	}

	fpPkHex := cwclient.BtcPkToHex(fpPk)

	res, err := wc.cwClient.QueryLastPubRandCommit(wc.Ctx(), fpPkHex)
	if err != nil {
		return nil, errors.Wrap(err, "QueryLastPubRandCommit failed")
	}

	commitmentBytes, err := hexutil.Decode(res.Commitment)
	if err != nil {
		return nil, errors.Wrapf(err, "decode commitment failed: %v", res.Commitment)
	}

	response := make(map[uint64]*finalitytypes.PubRandCommitResponse)
	response[res.StartHeight] = &finalitytypes.PubRandCommitResponse{
		NumPubRand: res.NumPubRand,
		Commitment: commitmentBytes,
	}

	return response, nil
}

// QueryBlock queries the block at the given height
func (wc *WasmContractController) QueryBlock(height uint64) (*types.BlockInfo, error) {
	res, err := wc.QueryBlocks(height, height, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query block by %v", height)
	}

	if len(res) != 1 {
		return nil, errors.Errorf("query blocks returned no block information for %v", height)
	}

	return res[0], nil
}

// QueryBlocks returns a list of blocks from startHeight to endHeight
func (wc *WasmContractController) QueryBlocks(startHeight, endHeight uint64, limit uint32) ([]*types.BlockInfo, error) {
	if endHeight < startHeight {
		return nil, errors.Errorf("the startHeight %v should not be higher than the endHeight %v", startHeight, endHeight)
	}
	count := endHeight - startHeight + 1
	if count > uint64(limit) {
		count = uint64(limit)
	}

	if count == 0 {
		wc.logger.Warn("QueryBlocks count is zero!")
		return nil, nil
	}

	res := make([]*types.BlockInfo, 0, count)
	for i := uint64(0); i < count; i++ {
		h := big.NewInt(int64(startHeight + i))
		header, err := wc.l2Client.HeaderByNumber(wc.Ctx(), h)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get header by number by %v", h)
		}

		hash := header.Hash()

		res = append(res, &types.BlockInfo{
			Height: h.Uint64(),
			Hash:   hash[:],
		})
	}

	return res, nil
}

// QueryBestBlock queries the tip block of the consumer chain
func (wc *WasmContractController) QueryBestBlock() (*types.BlockInfo, error) {
	safe, err := wc.l2Client.BlockReceipts(wc.Ctx(), rpc.BlockNumberOrHashWithNumber(rpc.SafeBlockNumber))
	if err != nil {
		return nil, errors.Wrap(err, "failed to QueryBestBlock by get SafeBlockNumber BlockReceipts")
	}

	if len(safe) == 0 {
		wc.logger.Warn("get QueryBestblock by get SafeBlockNumber no returns")
		return nil, nil
	}

	safeBlock := safe[0]
	hash := safeBlock.BlockHash

	return &types.BlockInfo{
		Height: safeBlock.BlockNumber.Uint64(),
		Hash:   hash[:],
	}, nil
}

// QueryActivatedHeight returns the activated height of the consumer chain
// error will be returned if the consumer chain has not been activated
func (wc *WasmContractController) QueryActivatedHeight() (uint64, error) {
	return wc.activatedHeight, nil
}

func (wc *WasmContractController) Close() error {
	return nil
}
