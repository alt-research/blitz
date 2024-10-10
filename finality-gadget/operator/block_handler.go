package operator

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"

	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/core/logging"
)

type IL2BlockProcesser interface {
	OnBlock(ctx context.Context, blk *types.Block) error
}

type L2BlockHandler struct {
	logger logging.Logger
	client *l2eth.L2EthClient

	latestBlockNumber  uint64
	latestBlockHash    common.Hash
	blockInterval      uint64
	fetchBlockInterval time.Duration

	processers      map[string]IL2BlockProcesser
	processersMutex sync.RWMutex

	wg sync.WaitGroup
}

func NewL2BlockHandler(
	ctx context.Context,
	logger logging.Logger,
	client *l2eth.L2EthClient) *L2BlockHandler {
	return &L2BlockHandler{
		logger:     logger.With("module", "l2BlockHandler"),
		processers: make(map[string]IL2BlockProcesser, 8),
		client:     client,
	}
}

func (h *L2BlockHandler) AddProcesser(name string, processer IL2BlockProcesser) {
	h.processersMutex.Lock()
	defer h.processersMutex.Unlock()

	h.logger.Info("add processer l2", "name", name)

	h.processers[name] = processer
}

func (h *L2BlockHandler) WithLatestBlock(number uint64, hash common.Hash) {
	h.logger.Info("latest block number", "number", number, "hash", hash)
	h.setProcessedBlock(number, hash)
}

func (h *L2BlockHandler) setProcessedBlock(number uint64, hash common.Hash) {
	h.latestBlockNumber = number
	h.latestBlockHash = hash
}

func (h *L2BlockHandler) initialize(ctx context.Context) {
	if h.fetchBlockInterval == 0 {
		h.fetchBlockInterval = 1 * time.Second
	}

	currentBlockNumber, err := h.client.BlockNumber(ctx)
	if err != nil {
		h.logger.Error("failed to get currently block number in boot", "err", err)
		// no return, it just for logger
	}

	h.logger.Info(
		"start block handler",
		"latest", h.latestBlockNumber,
		"current", currentBlockNumber,
		"fetchInterval", h.fetchBlockInterval,
		"blockInterval", h.blockInterval,
	)
}

func (h *L2BlockHandler) Start(ctx context.Context) {
	h.wg.Add(1)

	h.initialize(ctx)

	go func() {
		defer func() {
			h.logger.Info("Stop l2 block handler")
			h.wg.Done()
		}()

		h.logger.Info("Starting l2 block handler")

		ticker := time.NewTicker(h.fetchBlockInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.logger.Debug("on block handler ticker")
				err := h.fetchBlocks(ctx)
				if err != nil {
					h.logger.Error("fetch l2 block handler error", "err", err)
				}
			}
		}

	}()
}

func (h *L2BlockHandler) Wait() {
	h.wg.Wait()
}

func (h *L2BlockHandler) fetchBlocks(ctx context.Context) error {
	h.logger.Debug("fetch block", "latest", h.latestBlockNumber)

	currentBlockNumber, err := h.client.BlockNumber(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get current block number")
	}

	if currentBlockNumber <= h.latestBlockNumber {
		h.logger.Debug(
			"no need fetch block",
			"current", currentBlockNumber,
			"processed", h.latestBlockNumber)
		return nil
	}

	start := h.latestBlockNumber + 1
	for i := start; i < currentBlockNumber; i++ {
		if h.blockInterval > 1 {
			if i%h.blockInterval != 0 {
				h.logger.Debug(
					"skip block by no block interval",
					"number", i, "interval", h.blockInterval,
				)
				continue
			}
		}

		err := h.fetchBlock(ctx, i)
		if err != nil {
			return errors.Wrapf(err, "failed to fetch block %d", i)
		}
	}

	return nil
}

func (h *L2BlockHandler) fetchBlock(ctx context.Context, number uint64) error {
	h.logger.Debug("try fetch l2 block", "number", number)

	blk, err := h.client.BlockByNumber(ctx, big.NewInt(int64(number)))
	if err != nil {
		return errors.Wrapf(err, "failed to get l2 block by number %d", number)
	}

	if err := h.handleBlock(ctx, number, blk); err != nil {
		return errors.Wrapf(err, "handle block %d failed", number)
	}

	h.setProcessedBlock(number, blk.Hash())

	return nil
}

func (h *L2BlockHandler) handleBlock(ctx context.Context, number uint64, blk *types.Block) error {
	hash := blk.Hash()
	logger := h.logger.With("number", number, "hash", hash)
	logger.Info("handle l2 block")

	h.processersMutex.RLock()
	defer h.processersMutex.RUnlock()

	for n, i := range h.processers {
		logger.Debug("processer handle l2 block", "name", n)
		err := i.OnBlock(ctx, blk)
		if err != nil {
			logger.Error("processer handle l2 block faled", "name", n, "err", err)
		}
	}

	logger.Debug("handle l2 block stop")

	return nil
}
