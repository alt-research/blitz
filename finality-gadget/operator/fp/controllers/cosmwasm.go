package controllers

import (
	"context"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/babylonlabs-io/finality-provider/clientcontroller/api"
	"github.com/babylonlabs-io/finality-provider/clientcontroller/cosmwasm"
	cosmwasmcfg "github.com/babylonlabs-io/finality-provider/cosmwasmclient/config"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/babylonlabs-io/finality-provider/types"

	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
)

var _ api.ConsumerController = &CosmwasmConsumerController{}

type CosmwasmConsumerController struct {
	inner  *cosmwasm.CosmwasmConsumerController
	cfg    *fpcfg.CosmwasmConfig
	logger *zap.Logger

	ctx      context.Context
	l2Client *l2eth.L2EthClient
}

func NewCosmwasmConsumerController(
	ctx context.Context,
	cfg *configs.OperatorConfig,
	fpConfig *fpcfg.Config,
	zapLogger *zap.Logger) (*CosmwasmConsumerController, error) {
	wasmEncodingCfg := cosmwasmcfg.GetWasmdEncodingConfig()
	inner, err := cosmwasm.NewCosmwasmConsumerController(fpConfig.CosmwasmConfig, wasmEncodingCfg, zapLogger)
	if err != nil {
		return nil, errors.Wrap(err, "inner NewCosmwasmConsumerController failed")
	}

	l2Client, err := l2eth.NewL2EthClient(ctx, &cfg.Layer2)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create l2 eth client")
	}

	return &CosmwasmConsumerController{
		inner:    inner,
		cfg:      fpConfig.CosmwasmConfig,
		logger:   zapLogger,
		ctx:      ctx,
		l2Client: l2Client,
	}, nil
}

func (wc *CosmwasmConsumerController) Ctx() context.Context {
	return wc.ctx
}

// CommitPubRandList commits a list of EOTS public randomness the consumer chain
// it returns tx hash and error
func (wc *CosmwasmConsumerController) CommitPubRandList(
	fpPk *btcec.PublicKey,
	startHeight uint64,
	numPubRand uint64,
	commitment []byte,
	sig *schnorr.Signature) (*types.TxResponse, error) {
	return wc.inner.CommitPubRandList(fpPk, startHeight, numPubRand, commitment, sig)
}

// SubmitFinalitySig submits the finality signature to the consumer chain
func (wc *CosmwasmConsumerController) SubmitFinalitySig(
	fpPk *btcec.PublicKey,
	block *types.BlockInfo,
	pubRand *btcec.FieldVal,
	proof []byte,
	sig *btcec.ModNScalar) (*types.TxResponse, error) {
	return wc.inner.SubmitFinalitySig(fpPk, block, pubRand, proof, sig)
}

// SubmitBatchFinalitySigs submits a batch of finality signatures to the consumer chain
func (wc *CosmwasmConsumerController) SubmitBatchFinalitySigs(
	fpPk *btcec.PublicKey,
	blocks []*types.BlockInfo,
	pubRandList []*btcec.FieldVal,
	proofList [][]byte,
	sigs []*btcec.ModNScalar) (*types.TxResponse, error) {
	return wc.inner.SubmitBatchFinalitySigs(fpPk, blocks, pubRandList, proofList, sigs)
}

// Note: the following queries are only for PoC

// QueryFinalityProviderHasPower queries whether the finality provider has voting power at a given height
func (wc *CosmwasmConsumerController) QueryFinalityProviderHasPower(fpPk *btcec.PublicKey, blockHeight uint64) (bool, error) {
	return wc.inner.QueryFinalityProviderHasPower(fpPk, blockHeight)
}

// QueryLatestFinalizedBlock returns the latest finalized block
// Note: nil will be returned if the finalized block does not exist
func (wc *CosmwasmConsumerController) QueryLatestFinalizedBlock() (*types.BlockInfo, error) {
	return wc.inner.QueryLatestFinalizedBlock()
}

// QueryLastPublicRandCommit returns the last committed public randomness
func (wc *CosmwasmConsumerController) QueryLastPublicRandCommit(fpPk *btcec.PublicKey) (*types.PubRandCommit, error) {
	return wc.inner.QueryLastPublicRandCommit(fpPk)
}

// QueryBlock queries the block at the given height
func (wc *CosmwasmConsumerController) QueryBlock(height uint64) (*types.BlockInfo, error) {
	res, err := wc.QueryBlocks(height, height, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query block by %v", height)
	}

	if len(res) != 1 {
		return nil, errors.Errorf("query blocks returned no block information for %v", height)
	}

	return res[0], nil
}

// QueryIsBlockFinalized queries if the block at the given height is finalized
func (wc *CosmwasmConsumerController) QueryIsBlockFinalized(height uint64) (bool, error) {
	return wc.inner.QueryIsBlockFinalized(height)
}

// QueryBlocks returns a list of blocks from startHeight to endHeight
func (wc *CosmwasmConsumerController) QueryBlocks(startHeight, endHeight, limit uint64) ([]*types.BlockInfo, error) {
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

// QueryLatestBlockHeight queries the tip block height of the consumer chain
func (wc *CosmwasmConsumerController) QueryLatestBlockHeight() (uint64, error) {
	safe, err := wc.l2Client.BlockReceipts(wc.Ctx(), rpc.BlockNumberOrHashWithNumber(rpc.SafeBlockNumber))
	if err != nil {
		return 0, errors.Wrap(err, "failed to QueryBestBlock by get SafeBlockNumber BlockReceipts")
	}

	if len(safe) == 0 {
		wc.logger.Warn("get QueryBestblock by get SafeBlockNumber no returns")
		return 0, nil
	}

	safeBlock := safe[0]

	return safeBlock.BlockNumber.Uint64(), nil
}

// QueryActivatedHeight returns the activated height of the consumer chain
// error will be returned if the consumer chain has not been activated
func (wc *CosmwasmConsumerController) QueryActivatedHeight() (uint64, error) {
	return wc.inner.QueryActivatedHeight()
}

func (wc *CosmwasmConsumerController) Close() error {
	return wc.inner.Close()
}
