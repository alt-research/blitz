package operator

import (
	"context"
	"sync"

	"github.com/babylonlabs-io/finality-gadget/db"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	sdkClient "github.com/alt-research/blitz/finality-gadget/sdk/client"
)

type FinalityGadgetOperatorService struct {
	logger logging.Logger
	cfg    *configs.OperatorConfig

	l2Client             *l2eth.L2EthClient
	finalityGadgetClient sdkClient.IFinalityGadget

	wg sync.WaitGroup
}

func NewFinalityGadgetOperatorService(
	ctx context.Context,
	cfg *configs.OperatorConfig,
	logger logging.Logger,
	zapLogger *zap.Logger) (*FinalityGadgetOperatorService, error) {
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

	finalityGadgetClient, err := sdkClient.NewFinalityGadget(cfg.Babylon.FinalityGadget(), db, zapLogger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create babylon client")
	}

	return &FinalityGadgetOperatorService{
		logger: logger,
		cfg:    cfg,

		l2Client:             l2Client,
		finalityGadgetClient: finalityGadgetClient,
	}, nil
}

func (s *FinalityGadgetOperatorService) Start(ctx context.Context) error {
	s.wg.Add(1)
	defer func() {
		s.logger.Info("Stop finality gadget operator service", "name", s.cfg.Common.Name)
		s.wg.Done()
	}()

	s.logger.Info("Starting finality gadget operator service", "name", s.cfg.Common.Name)

	for {
		select {
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *FinalityGadgetOperatorService) Wait() {
	s.wg.Wait()
}
