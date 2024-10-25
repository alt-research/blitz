package controllers

import (
	"context"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/babylonlabs-io/finality-gadget/db"
	"github.com/babylonlabs-io/finality-provider/clientcontroller/api"
	"github.com/babylonlabs-io/finality-provider/clientcontroller/opstackl2"
	cwcclient "github.com/babylonlabs-io/finality-provider/cosmwasmclient/client"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/babylonlabs-io/finality-provider/types"

	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	sdkClient "github.com/alt-research/blitz/finality-gadget/sdk/client"
)

var _ api.ConsumerController = &OrbitConsumerController{}

type OrbitConsumerController struct {
	inner  *opstackl2.OPStackL2ConsumerController
	cfg    *fpcfg.OPStackL2Config
	logger *zap.Logger

	ctx                  context.Context
	finalityGadgetClient sdkClient.IFinalityGadget
	cwClient             *cwcclient.Client
	l2Client             *l2eth.L2EthClient

	activeHeight    uint64
	backHeightCount uint64
}

func NewOrbitConsumerController(
	ctx context.Context,
	cfg *configs.OperatorConfig,
	fpConfig *fpcfg.Config,
	zapLogger *zap.Logger,
) (*OrbitConsumerController, error) {
	inner, err := opstackl2.NewOPStackL2ConsumerController(fpConfig.OPStackL2Config, zapLogger)
	if err != nil {
		return nil, errors.Wrap(err, "inner NewOrbitConsumerController failed")
	}

	l2Client, err := l2eth.NewL2EthClient(ctx, &cfg.Layer2)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create l2 eth client")
	}

	// Init local DB for storing and querying blocks
	db, err := db.NewBBoltHandler(cfg.Babylon.FinalityGadget().DBFilePath, zapLogger)
	if err != nil {
		return nil, errors.Errorf("failed to create DB handler: %w", err)
	}
	defer db.Close()
	err = db.CreateInitialSchema()
	if err != nil {
		return nil, errors.Errorf("create initial buckets error: %w", err)
	}

	finalityGadgetClient, err := sdkClient.NewFinalityGadget(
		cfg.Babylon.FinalityGadget(),
		db,
		zapLogger,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create babylon client")
	}

	return &OrbitConsumerController{
		inner:                inner,
		cfg:                  fpConfig.OPStackL2Config,
		logger:               zapLogger,
		ctx:                  ctx,
		cwClient:             inner.CwClient,
		finalityGadgetClient: finalityGadgetClient,
		l2Client:             l2Client,
		activeHeight:         cfg.Layer2.ActivatedHeight,
		backHeightCount:      cfg.Layer2.BackHeightCount,
	}, nil
}

func (wc *OrbitConsumerController) Ctx() context.Context {
	return wc.ctx
}

// CommitPubRandList commits a list of EOTS public randomness the consumer chain
// it returns tx hash and error
func (wc *OrbitConsumerController) CommitPubRandList(
	fpPk *btcec.PublicKey,
	startHeight uint64,
	numPubRand uint64,
	commitment []byte,
	sig *schnorr.Signature) (*types.TxResponse, error) {
	wc.logger.Sugar().Debugf("CommitPubRandList %v", startHeight)
	return wc.inner.CommitPubRandList(fpPk, startHeight, numPubRand, commitment, sig)
}

// SubmitBatchFinalitySigs submits a batch of finality signatures to the consumer chain
func (wc *OrbitConsumerController) SubmitBatchFinalitySigs(
	fpPk *btcec.PublicKey,
	blocks []*types.BlockInfo,
	pubRandList []*btcec.FieldVal,
	proofList [][]byte,
	sigs []*btcec.ModNScalar) (*types.TxResponse, error) {
	wc.logger.Sugar().Debugf("SubmitBatchFinalitySigs %v", blocks)
	tx, err := wc.submitBatchFinalitySigs(fpPk, blocks, pubRandList, proofList, sigs)
	if err != nil {
		wc.logger.Sugar().Errorf("SubmitFinalitySig %v failed: %v", blocks, err)
	}
	return tx, err
}

// Note: the following queries are only for PoC

// QueryFinalityProviderHasPower queries whether the finality provider has voting power at a given height
func (wc *OrbitConsumerController) QueryFinalityProviderHasPower(fpPk *btcec.PublicKey, blockHeight uint64) (bool, error) {
	//return wc.inner.QueryFinalityProviderHasPower(fpPk, blockHeight)
	return true, nil
}

// QueryLatestFinalizedBlock returns the latest finalized block
// Note: nil will be returned if the finalized block does not exist
func (wc *OrbitConsumerController) QueryLatestFinalizedBlock() (*types.BlockInfo, error) {
	logger := wc.logger.Sugar()
	logger.Debugf("QueryLatestFinalizedBlock")

	res, err := wc.queryBlock(rpc.BlockNumberOrHashWithNumber(rpc.FinalizedBlockNumber))

	if err != nil {
		logger.Errorf("QueryLatestFinalizedBlock failed by %v", err)
	} else {
		logger.Debugf("QueryLatestFinalizedBlock res %v", res)
	}

	return res, err
}

// QueryLastPublicRandCommit returns the last committed public randomness
func (wc *OrbitConsumerController) QueryLastPublicRandCommit(fpPk *btcec.PublicKey) (*types.PubRandCommit, error) {
	res, err := wc.inner.QueryLastPublicRandCommit(fpPk)

	wc.logger.Sugar().Debugf("QueryLastPublicRandCommit res %v", res)

	return res, err
}

// QueryBlock queries the block at the given height
func (wc *OrbitConsumerController) QueryBlock(height uint64) (*types.BlockInfo, error) {
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
func (wc *OrbitConsumerController) QueryIsBlockFinalized(height uint64) (bool, error) {
	return wc.inner.QueryIsBlockFinalized(height)
}

// QueryBlocks returns a list of blocks from startHeight to endHeight
func (wc *OrbitConsumerController) QueryBlocks(startHeight, endHeight, limit uint64) ([]*types.BlockInfo, error) {
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
func (wc *OrbitConsumerController) QueryLatestBlockHeight() (uint64, error) {
	logger := wc.logger.Sugar()
	logger.Debugf("QueryLatestBlockHeight")

	res, err := wc.queryBlock(rpc.BlockNumberOrHashWithNumber(rpc.SafeBlockNumber))
	height := res.Height
	if height <= wc.backHeightCount {
		height = 1
	} else {
		height = height - wc.backHeightCount
	}

	if err != nil {
		logger.Errorf("QueryLatestBlockHeight failed by %v", err)
	} else {
		logger.Debugf("QueryLatestBlockHeight res %v", height)
	}

	return height, err
}

// QueryActivatedHeight returns the activated height of the consumer chain
// error will be returned if the consumer chain has not been activated
func (wc *OrbitConsumerController) QueryActivatedHeight() (uint64, error) {
	return wc.activeHeight, nil
}

func (wc *OrbitConsumerController) Close() error {
	return wc.inner.Close()
}
