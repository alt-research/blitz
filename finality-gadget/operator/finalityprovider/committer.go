package finalityprovider

import (
	"context"
	"fmt"
	"sync"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	fpeotsmanager "github.com/babylonlabs-io/finality-provider/eotsmanager"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/operator/finalityprovider/cwclient"
	sdkClient "github.com/alt-research/blitz/finality-gadget/sdk/client"
)

type L2BlockCommitter struct {
	logger logging.Logger
	cfg    *Config
	BtcPk  *btcec.PublicKey

	pendingL2Blocks map[uint64]*types.Block

	// the numbers for blocks to be committed:
	//  `committing` <-- `minPending` <- `maxPending` <- `committed`
	//
	maxPendingNumber uint64
	// the min pending blocks
	minPendingNumber uint64
	// if not zero, means currently had some block to commit, it should be
	// zero or minPendingNumber
	committingNumber uint64
	// the finalizedNumber, should always be > `minPendingNumber` and `committingNumber`
	finalizedNumber uint64
	// the activatedHeight
	activatedHeight uint64

	finalityGadgetClient sdkClient.IFinalityGadget
	cwClient             cwclient.ICosmosWasmContractClient
	em                   fpeotsmanager.EOTSManager

	mu sync.Mutex
}

func NewL2BlockCommitter(
	logger logging.Logger,
	cfg *Config,
	finalityGadgetClient sdkClient.IFinalityGadget,
	cwClient cwclient.ICosmosWasmContractClient,
	activatedHeight uint64,
	BtcPk *btcec.PublicKey,
) *L2BlockCommitter {
	return &L2BlockCommitter{
		logger:               logger.With("module", "l2BlockCommitter"),
		cfg:                  cfg,
		finalityGadgetClient: finalityGadgetClient,
		cwClient:             cwClient,
		activatedHeight:      activatedHeight,
		pendingL2Blocks:      make(map[uint64]*types.Block, 32),
	}
}

func (c *L2BlockCommitter) Append(blk *types.Block) {
	c.mu.Lock()
	defer c.mu.Unlock()

	number := blk.NumberU64()

	logger := c.logger.With("number", number, "hash", blk.Hash())
	logger.Info("append new l2 block")

	if number <= c.activatedHeight {
		return
	}

	_, ok := c.pendingL2Blocks[number]
	if ok {
		logger.Debug("l2 block had append into the committer")
		return
	}

	if c.finalizedNumber > number {
		logger.Debug("l2 block had committed", "finalized", c.finalizedNumber)
		return
	}

	// upgrade the max-min
	if c.maxPendingNumber < number {
		c.maxPendingNumber = number
	}

	if c.minPendingNumber > number {
		c.minPendingNumber = number
	}

	// on initialization
	if c.minPendingNumber == 0 {
		if len(c.pendingL2Blocks) == 0 {
			c.minPendingNumber = number
		}
	}

	c.pendingL2Blocks[number] = blk

	logger.Debug(
		"append new l2 block",
		"len", len(c.pendingL2Blocks),
		"min", c.minPendingNumber,
		"max", c.maxPendingNumber,
		"committing", c.committingNumber,
		"finalized", c.finalizedNumber,
	)
}

func (c *L2BlockCommitter) tryGetBlockToCommit() *types.Block {
	// no pending blocks
	if len(c.pendingL2Blocks) == 0 {
		return nil
	}

	if c.committingNumber != 0 {
		if c.committingNumber >= c.minPendingNumber {
			// no need commit
			c.logger.Debug(
				"tryGetBlockToCommit not need commit by is committing",
				"committing", c.committingNumber,
				"pending", c.minPendingNumber,
			)

			return nil
		}
	}

	if c.minPendingNumber == 0 {
		c.logger.Warn(
			"tryGetBlockToCommit min pending number is 0 but pending blocks not empty",
			"len", len(c.pendingL2Blocks),
			"min", c.minPendingNumber,
			"max", c.maxPendingNumber,
		)

		return nil
	}

	blk, ok := c.pendingL2Blocks[c.minPendingNumber]
	if !ok {
		c.logger.Error(
			"tryGetBlockToCommit not found pending block",
			"len", len(c.pendingL2Blocks),
			"min", c.minPendingNumber,
			"max", c.maxPendingNumber,
		)

		return nil
	}

	return blk
}

// onFinalityBlock when got a new Finalized block, we can delete all datas useless
func (c *L2BlockCommitter) onFinalizedBlock(number uint64, hash common.Hash) {
	logger := c.logger.With("number", number, "hash", hash)
	logger.Info(
		"on Finalized l2 block",
		"min", c.minPendingNumber,
		"max", c.maxPendingNumber)

	if number < c.minPendingNumber {
		return
	}

	if number > c.maxPendingNumber {
		// clean all datas

	}

	blk, ok := c.pendingL2Blocks[number]
	if ok {
		pendingHash := blk.Hash()
		if pendingHash != hash {
			// the hash is invalid, means our full node may had some error
			// so we only need panic to exit the service.
			panic(
				fmt.Sprintf(
					"the final hash is not equal to the local hash for block %v, local %v, final %v",
					number,
					pendingHash.String(),
					hash.String(),
				),
			)
		}
	}

	newPending := make(map[uint64]*types.Block, len(c.pendingL2Blocks))
	for n, pending := range c.pendingL2Blocks {
		if n >= number {
			newPending[n] = pending
		}
	}
	c.pendingL2Blocks = newPending
	c.minPendingNumber = number
	c.finalizedNumber = number

	if c.committingNumber != 0 {
		logger.Debug(
			"committing block had finalized",
			"committing", c.committingNumber,
			"number", number)
		c.committingNumber = 0
	}

	logger.Debug(
		"on finality l2 block to",
		"len", len(c.pendingL2Blocks),
		"min", c.minPendingNumber,
		"max", c.maxPendingNumber,
		"committing", c.committingNumber,
		"finalized", c.finalizedNumber,
	)
}

func (c *L2BlockCommitter) dropFinalizedBlock(ctx context.Context) error {
	maxFinalizedBlock := uint64(0)
	maxFinalizedBlockHash := common.Hash{}

	for i := c.minPendingNumber; i <= c.maxPendingNumber; i++ {
		blk, ok := c.pendingL2Blocks[i]
		if ok {
			blockInfo := ToFgBlock(blk)
			c.logger.Debug(
				"QueryIsBlockBabylonFinalized",
				"hash", blockInfo.BlockHash,
				"height", blockInfo.BlockHeight,
				"timestamp", blockInfo.BlockTimestamp,
			)

			isFinalized, err := c.finalityGadgetClient.QueryIsBlockBabylonFinalized(
				ctx,
				blockInfo,
			)
			if err != nil {
				return errors.Wrapf(err, "failed to QueryIsBlockBabylonFinalized by %d", i)
			}

			c.logger.Debug("QueryIsBlockBabylonFinalized", "number", i, "isFinalized", isFinalized)

			if isFinalized {
				maxFinalizedBlock = i
				maxFinalizedBlockHash = blk.Hash()
			}

		} else {
			c.logger.Warn("not found block in pending", "number", i)
		}
	}

	if maxFinalizedBlock != 0 {
		c.logger.Debug("cleaning up", "number", maxFinalizedBlock)

		c.onFinalizedBlock(maxFinalizedBlock, maxFinalizedBlockHash)
	}

	return nil
}

func (c *L2BlockCommitter) TryCommitPendingBlock(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.pendingL2Blocks) == 0 && c.minPendingNumber == 0 {
		// no append
		return nil
	}

	c.logger.Debug(
		"try to commit block",
		"len", len(c.pendingL2Blocks),
		"min", c.minPendingNumber,
		"max", c.maxPendingNumber,
		"committing", c.committingNumber,
		"finalized", c.finalizedNumber,
	)

	if err := c.dropFinalizedBlock(ctx); err != nil {
		return errors.Wrap(err, "dropFinalizedBlock failed when trying to commit")
	}

	blk := c.tryGetBlockToCommit()
	if blk != nil {
		err := c.commitBlock(ctx, blk)
		if err != nil {
			return errors.Wrapf(err, "failed to commit l2 block %v", blk.NumberU64())
		} else {
			c.committingNumber = blk.NumberU64()
		}
	}

	return nil
}

func (c *L2BlockCommitter) commitBlock(ctx context.Context, blk *types.Block) error {
	number := blk.NumberU64()
	c.logger.Info("commit block", "number", number, "hash", blk.Hash())

	err := c.CommitPublicRandomness(ctx, number)
	if err != nil {
		return errors.Wrapf(err, "commit public random failed with %d", number)
	}

	return nil
}
