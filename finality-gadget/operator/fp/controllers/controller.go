package controllers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	bbnclient "github.com/babylonlabs-io/babylon/v3/client/client"
	"github.com/babylonlabs-io/finality-gadget/db"
	"github.com/babylonlabs-io/finality-provider/bsn/rollup/clientcontroller"
	rollupfpconfig "github.com/babylonlabs-io/finality-provider/bsn/rollup/config"
	"github.com/babylonlabs-io/finality-provider/clientcontroller/api"
	"github.com/babylonlabs-io/finality-provider/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"

	"github.com/alt-research/blitz/finality-gadget/metrics"
)

var _ api.ConsumerController = &OrbitConsumerController{}

type OrbitConsumerController struct {
	logger *zap.Logger

	l2Client *l2eth.L2EthClient

	fpConfig *rollupfpconfig.RollupFPConfig
	*clientcontroller.RollupBSNController
	blitzMetrics *metrics.FpMetrics
	metricsMu    sync.Mutex

	backHeightCount uint64

	bbnClient *bbnclient.Client
}

func NewOrbitConsumerController(
	ctx context.Context,
	cfg *configs.OperatorConfig,
	fpConfig *rollupfpconfig.RollupFPConfig,
	blitzMetrics *metrics.FpMetrics,
	zapLogger *zap.Logger,
) (*OrbitConsumerController, error) {
	consumerCon, err := clientcontroller.NewRollupBSNController(fpConfig, zapLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create rpc client for the consumer chain rollup: %w", err)
	}

	babylonConfig := fpConfig.GetBabylonConfig()
	if err := babylonConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config for Babylon client: %w", err)
	}

	bc, err := bbnclient.New(
		&babylonConfig,
		zapLogger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Babylon client: %w", err)
	}

	// Init local DB for storing and querying blocks
	db, err := db.NewBBoltHandler(cfg.Babylon.FinalityGadgetCfg.DBFilePath, zapLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create DB handler: %w", err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			zapLogger.Sugar().Errorf("failed to close db: %v", err)
		}
	}()
	err = db.CreateInitialSchema()
	if err != nil {
		return nil, fmt.Errorf("create initial buckets error: %w", err)
	}

	res := &OrbitConsumerController{
		RollupBSNController: consumerCon,
		bbnClient:           bc,
		blitzMetrics:        blitzMetrics,
		fpConfig:            fpConfig,
		logger:              zapLogger,
		backHeightCount:     cfg.Layer2.BackHeightCount,
	}

	go func() {
		res.logger.Info("Starting fp token metrics")

		res.recordFpBalance(ctx)

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				res.logger.Debug("on recordAddressToken ticker")
				res.recordFpBalance(ctx)
			}
		}
	}()

	return res, nil
}

// CommitPubRandList commits a list of EOTS public randomness the consumer chain
// it returns tx hash and error
func (wc *OrbitConsumerController) CommitPubRandList(
	ctx context.Context, req *api.CommitPubRandListRequest) (*types.TxResponse, error) {
	wc.logger.Sugar().Debugf(
		"CommitPubRandList %v %v",
		req.StartHeight, req.NumPubRand)
	resp, err := wc.RollupBSNController.CommitPubRandList(ctx, req)
	if err != nil {
		return nil, err
	}

	wc.recordFpBalance(ctx)
	return resp, nil
}

// SubmitBatchFinalitySigs submits a batch of finality signatures to the consumer chain
func (wc *OrbitConsumerController) SubmitBatchFinalitySigs(
	ctx context.Context, req *api.SubmitBatchFinalitySigsRequest) (*types.TxResponse, error) {
	wc.logger.Sugar().Debugf("SubmitBatchFinalitySigs %v", req.Blocks)

	resp, err := wc.RollupBSNController.SubmitBatchFinalitySigs(ctx, req)
	if err != nil {
		return nil, err
	}

	wc.recordFpBalance(ctx)
	return resp, nil
}

func (wc *OrbitConsumerController) recordFpBalance(ctxBase context.Context) {
	go func() {
		wc.metricsMu.Lock()
		defer wc.metricsMu.Unlock()

		ctx, cancel := context.WithTimeout(ctxBase, 8*time.Second)
		defer cancel()

		address, err := wc.bbnClient.GetAddr()
		if err != nil {
			wc.logger.Sugar().Errorw("recordFpBalance failed to get addr", "err", err)
			return
		}

		balance, err := wc.queryBalance(ctx, address, "ubbn")
		if err != nil {
			wc.logger.Sugar().Errorw("recordFpBalance failed to get balance by addr", "addr", address, "err", err)
		}

		wc.logger.Sugar().Debugw("record fp balance", "addr", address, "balance", balance)
		wc.blitzMetrics.RecordFpBalance(address, balance)
	}()
}

// QueryBalances returns balances at the address.
func (wc *OrbitConsumerController) queryBalance(ctx context.Context, address, denom string) (float64, error) {
	req := banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   denom,
	}

	data, err := req.Marshal()
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	result, err := wc.bbnClient.RPCClient.ABCIQuery(ctx, "/cosmos.bank.v1beta1.Query/Balance", data)
	if err != nil {
		return 0, fmt.Errorf("failed to query balance: %w", err)
	}

	var balancesResp banktypes.QueryBalanceResponse
	if err := balancesResp.Unmarshal(result.Response.Value); err != nil {
		return 0, fmt.Errorf("failed to unmarshal balance response: %w", err)
	}

	// from ubbn to bbn
	balanceUBBN := balancesResp.GetBalance().Amount.BigInt()
	return float64(balanceUBBN.Uint64()) / 1000000, nil
}

func (wc *OrbitConsumerController) Close() error {
	wc.logger.Sugar().Debugw("close OrbitConsumerController")
	wc.l2Client.Close()

	if wc.bbnClient.IsRunning() {
		if err := wc.bbnClient.Stop(); err != nil {
			return fmt.Errorf("failed to stop Babylon client: %w", err)
		}
	}

	return wc.RollupBSNController.Close()
}
