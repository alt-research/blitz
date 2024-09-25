package operator

import (
	"context"
	"sync"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
)

type FinalityGadgetOperatorService struct {
	logger logging.Logger
	cfg    *configs.OperatorConfig

	wg sync.WaitGroup
}

func NewFinalityGadgetOperatorService(
	ctx context.Context,
	cfg *configs.OperatorConfig,
	logger logging.Logger) (*FinalityGadgetOperatorService, error) {
	return &FinalityGadgetOperatorService{
		logger: logger,
		cfg:    cfg,
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
