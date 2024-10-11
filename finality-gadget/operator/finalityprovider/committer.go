package finalityprovider

import (
	"sync"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/ethereum/go-ethereum/core/types"
)

type L2BlockCommitter struct {
	logger logging.Logger

	pendingL2Blocks  map[uint64]*types.Block
	maxPendingNumber uint64
	minPendingNumber uint64
	committingNumber uint64
	committedNumber  uint64

	blocksMutex sync.RWMutex
}

func NewL2BlockCommitter(logger logging.Logger) *L2BlockCommitter {
	return &L2BlockCommitter{
		logger:          logger.With("module", "l2BlockCommitter"),
		pendingL2Blocks: make(map[uint64]*types.Block, 32),
	}
}

func (c *L2BlockCommitter) append(blk *types.Block) {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()

	number := blk.NumberU64()

	logger := c.logger.With("number", number, "hash", blk.Hash())
	logger.Info("append new l2 block")

	_, ok := c.pendingL2Blocks[number]
	if ok {
		logger.Debug("l2 block had append into the committer")
		return
	}

	if c.committedNumber > number {
		logger.Debug("l2 block had committed", "committed", c.committedNumber)
		return
	}

	if c.maxPendingNumber < number {
		c.maxPendingNumber = number
	}

	if c.minPendingNumber > number {
		c.minPendingNumber = number
	}

	c.pendingL2Blocks[number] = blk

	logger.Debug(
		"append new l2 block",
		"len", len(c.pendingL2Blocks),
		"min", c.minPendingNumber,
		"max", c.maxPendingNumber,
		"committing", c.committingNumber,
		"committed", c.committedNumber,
	)
}

func (c *L2BlockCommitter) tryCommit() error {

	return nil
}
