package controllers

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/babylonlabs-io/finality-gadget/db"
	"github.com/babylonlabs-io/finality-provider/bsn/rollup/clientcontroller"
	rollupfpconfig "github.com/babylonlabs-io/finality-provider/bsn/rollup/config"
	"github.com/babylonlabs-io/finality-provider/clientcontroller/api"
	"github.com/babylonlabs-io/finality-provider/types"

	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
)

var _ api.ConsumerController = &OrbitConsumerController{}

type OrbitConsumerController struct {
	logger *zap.Logger

	ctx      context.Context
	l2Client *l2eth.L2EthClient

	fpConfig *rollupfpconfig.RollupFPConfig
	*clientcontroller.RollupBSNController

	activeHeight    uint64
	backHeightCount uint64
}

func NewOrbitConsumerController(
	ctx context.Context,
	cfg *configs.OperatorConfig,
	fpConfig *rollupfpconfig.RollupFPConfig,
	zapLogger *zap.Logger,
) (*OrbitConsumerController, error) {
	consumerCon, err := clientcontroller.NewRollupBSNController(fpConfig, zapLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create rpc client for the consumer chain rollup: %w", err)
	}

	// Init local DB for storing and querying blocks
	db, err := db.NewBBoltHandler(cfg.Babylon.FinalityGadgetCfg.DBFilePath, zapLogger)
	if err != nil {
		return nil, errors.Errorf("failed to create DB handler: %w", err)
	}
	defer db.Close()
	err = db.CreateInitialSchema()
	if err != nil {
		return nil, errors.Errorf("create initial buckets error: %w", err)
	}

	return &OrbitConsumerController{
		RollupBSNController: consumerCon,
		fpConfig:            fpConfig,
		logger:              zapLogger,
		ctx:                 ctx,
		backHeightCount:     cfg.Layer2.BackHeightCount,
	}, nil
}

func (wc *OrbitConsumerController) Ctx() context.Context {
	return wc.ctx
}

// CommitPubRandList commits a list of EOTS public randomness the consumer chain
// it returns tx hash and error
func (wc *OrbitConsumerController) CommitPubRandList(
	ctx context.Context, req *api.CommitPubRandListRequest) (*types.TxResponse, error) {
	wc.logger.Sugar().Debugf(
		"CommitPubRandList %v %v",
		req.StartHeight, req.NumPubRand)
	return wc.RollupBSNController.CommitPubRandList(wc.ctx, req)
}

// SubmitBatchFinalitySigs submits a batch of finality signatures to the consumer chain
func (wc *OrbitConsumerController) SubmitBatchFinalitySigs(
	ctx context.Context, req *api.SubmitBatchFinalitySigsRequest) (*types.TxResponse, error) {
	wc.logger.Sugar().Debugf("SubmitBatchFinalitySigs %v", req.Blocks)
	return wc.RollupBSNController.SubmitBatchFinalitySigs(ctx, req)
}

func (wc *OrbitConsumerController) Close() error {
	wc.l2Client.Close()
	return wc.RollupBSNController.Close()
}
