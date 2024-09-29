package operator

import (
	"context"
	"sync"

	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/core/logging"
)

type L2BlockHandler struct {
	logger logging.Logger
	client *l2eth.L2EthClient

	wg sync.WaitGroup
}

func NewL2BlockHandler(
	ctx context.Context,
	logger logging.Logger,
	client *l2eth.L2EthClient) *L2BlockHandler {
	return &L2BlockHandler{
		logger: logger.With("module", "l2BlockHandler"),
		client: client,
	}
}

func (h *L2BlockHandler) Start(ctx context.Context) {
	h.wg.Add(1)

	go func() {
		defer func() {
			h.logger.Info("Stop l2 block handler")
			h.wg.Done()
		}()

		h.logger.Info("Starting l2 block handler")

		for {
			select {
			case <-ctx.Done():
				return
			}
		}

	}()
}

func (h *L2BlockHandler) Wait() {
	h.wg.Wait()
}
