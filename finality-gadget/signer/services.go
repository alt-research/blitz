package signer

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/signer/configs"
)

type FinalityGadgetSignerService struct {
	logger logging.Logger
	cfg    *configs.SignerConfig

	l2Client *l2eth.L2EthClient

	wg sync.WaitGroup
}

func NewFinalityGadgetSignerService(
	ctx context.Context,
	cfg *configs.SignerConfig,
	logger logging.Logger) (*FinalityGadgetSignerService, error) {
	l2Client, err := l2eth.NewL2EthClient(ctx, &cfg.Layer2)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create l2 eth client")
	}

	return &FinalityGadgetSignerService{
		logger: logger,
		cfg:    cfg,

		l2Client: l2Client,
	}, nil
}

func (s *FinalityGadgetSignerService) Start(ctx context.Context) error {
	s.wg.Add(1)
	defer func() {
		s.logger.Info("Stop finality gadget signer service", "name", s.cfg.Common.Name)
		s.wg.Done()
	}()

	s.logger.Info("Starting finality gadget signer service", "name", s.cfg.Common.Name)

	for {
		select {
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *FinalityGadgetSignerService) Wait() {
	s.wg.Wait()
}
