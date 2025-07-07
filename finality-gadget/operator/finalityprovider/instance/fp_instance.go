package fp_instance

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	bbntypes "github.com/babylonlabs-io/babylon/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	fpcc "github.com/babylonlabs-io/finality-provider/clientcontroller"
	ccapi "github.com/babylonlabs-io/finality-provider/clientcontroller/api"
	"github.com/babylonlabs-io/finality-provider/eotsmanager"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/babylonlabs-io/finality-provider/finality-provider/proto"
	"github.com/babylonlabs-io/finality-provider/finality-provider/service"
	"github.com/babylonlabs-io/finality-provider/finality-provider/store"
	fpmetrics "github.com/babylonlabs-io/finality-provider/metrics"
	"github.com/babylonlabs-io/finality-provider/types"

	"github.com/alt-research/blitz/finality-gadget/metrics"
	finalitygadget "github.com/alt-research/blitz/finality-gadget/sdk/client"
)

type FinalityProviderInstance struct {
	btcPk *bbntypes.BIP340PubKey

	cfg *fpcfg.Config

	logger      *zap.Logger
	consumerCon ccapi.ConsumerController
	poller      *service.ChainPoller
	metrics     *fpmetrics.FpMetrics

	criticalErrChan chan<- *CriticalError

	isStarted *atomic.Bool

	blitzMetrics *metrics.FpMetrics
	cwClient     finalitygadget.ICosmWasmClient

	inner *service.FinalityProviderInstance

	wg   sync.WaitGroup
	quit chan struct{}
}

// NewFinalityProviderInstance returns a FinalityProviderInstance instance with the given Babylon public key
// the finality-provider should be registered before
func NewFinalityProviderInstance(
	ctx context.Context,
	blitzMetrics *metrics.FpMetrics,
	fpPk *bbntypes.BIP340PubKey,
	cfg *fpcfg.Config,
	s *store.FinalityProviderStore,
	prStore *store.PubRandProofStore,
	cc ccapi.ClientController,
	consumerCon ccapi.ConsumerController,
	em eotsmanager.EOTSManager,
	metrics *fpmetrics.FpMetrics,
	errChan chan<- *CriticalError,
	cwClient finalitygadget.ICosmWasmClient,
	logger *zap.Logger,
) (*FinalityProviderInstance, error) {
	sfp, err := s.GetFinalityProvider(fpPk.MustToBTCPK())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the finality provider %s from DB: %w", fpPk.MarshalHex(), err)
	}

	if sfp.Status == proto.FinalityProviderStatus_SLASHED {
		return nil, fmt.Errorf("the finality provider instance is already slashed")
	}

	errChanInner := NewCriticalErrorChan(ctx, logger, errChan)

	inner, err := service.NewFinalityProviderInstance(
		fpPk, cfg, s, prStore,
		cc, consumerCon,
		em, metrics,
		errChanInner, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create the finality provider instance inner by %v", err)
	}

	return &FinalityProviderInstance{
		btcPk:           bbntypes.NewBIP340PubKeyFromBTCPK(sfp.BtcPk),
		cfg:             cfg,
		logger:          logger,
		isStarted:       atomic.NewBool(false),
		criticalErrChan: errChan,
		consumerCon:     consumerCon,
		metrics:         metrics,
		blitzMetrics:    blitzMetrics,
		cwClient:        cwClient,
		inner:           inner,
	}, nil
}

func (fp *FinalityProviderInstance) Start() error {
	if fp.isStarted.Swap(true) {
		return fmt.Errorf("the finality-provider instance %s is already started", fp.GetBtcPkHex())
	}

	if fp.IsJailed() {
		fp.logger.Warn("the finality provider is jailed",
			zap.String("pk", fp.GetBtcPkHex()))
	}

	startHeight, err := fp.DetermineStartHeight()
	if err != nil {
		return fmt.Errorf("failed to get the start height: %w", err)
	}

	fp.logger.Info("starting the finality provider instance",
		zap.String("pk", fp.GetBtcPkHex()), zap.Uint64("height", startHeight))

	poller := service.NewChainPoller(fp.logger, fp.cfg.PollerConfig, fp.consumerCon, fp.metrics)

	if err := poller.Start(startHeight); err != nil {
		return fmt.Errorf("failed to start the poller with start height %d: %w", startHeight, err)
	}

	fp.poller = poller
	fp.quit = make(chan struct{})

	fp.wg.Add(2)
	go fp.finalitySigSubmissionLoop()
	go fp.randomnessCommitmentLoop()

	return nil
}

func (fp *FinalityProviderInstance) Stop() error {
	if !fp.isStarted.Swap(false) {
		return fmt.Errorf("the finality-provider %s has already stopped", fp.GetBtcPkHex())
	}

	if err := fp.poller.Stop(); err != nil {
		return fmt.Errorf("failed to stop the poller: %w", err)
	}

	fp.logger.Info("stopping finality-provider instance", zap.String("pk", fp.GetBtcPkHex()))

	close(fp.quit)
	fp.wg.Wait()

	fp.logger.Info("the finality-provider instance is successfully stopped", zap.String("pk", fp.GetBtcPkHex()))

	return nil
}

func (fp *FinalityProviderInstance) GetConfig() *fpcfg.Config {
	return fp.inner.GetConfig()
}

func (fp *FinalityProviderInstance) IsRunning() bool {
	return fp.isStarted.Load()
}

// IsJailed returns true if fp is JAILED
// NOTE: it retrieves the the status from the db to
// ensure status is up-to-date
func (fp *FinalityProviderInstance) IsJailed() bool {
	return fp.inner.IsJailed()
}

func (fp *FinalityProviderInstance) finalitySigSubmissionLoop() {
	defer fp.wg.Done()

	for {
		select {
		case <-time.After(fp.cfg.SignatureSubmissionInterval):
			// start submission in the first iteration
			pollerBlocks := fp.getBatchBlocksFromChan()
			if len(pollerBlocks) == 0 {
				continue
			}

			if fp.IsJailed() {
				fp.logger.Warn("the finality-provider is jailed",
					zap.String("pk", fp.GetBtcPkHex()),
				)

				continue
			}

			targetHeight := pollerBlocks[len(pollerBlocks)-1].Height
			fp.logger.Debug("the finality-provider received new block(s), start processing",
				zap.String("pk", fp.GetBtcPkHex()),
				zap.Uint64("start_height", pollerBlocks[0].Height),
				zap.Uint64("end_height", targetHeight),
			)

			// check if the public randomness proof had committed
			lastPublicRandCommit, err := fp.consumerCon.QueryLastPublicRandCommit(fp.GetBtcPk())
			if err != nil {
				fp.logger.Sugar().Error("QueryLastPublicRandCommit failed", "err", err.Error())
				continue
			}

			if lastPublicRandCommit == nil {
				fp.logger.Warn("not found last publicRandCommit",
					zap.String("pk", fp.GetBtcPkHex()),
				)
				continue
			}

			if lastPublicRandCommit.EndHeight() < targetHeight {
				fp.logger.Warn("the last publicRandCommit end height not cover the target height",
					zap.Uint64("rand_commit_start_height", lastPublicRandCommit.StartHeight),
					zap.Uint64("rand_commit_end_height", lastPublicRandCommit.EndHeight()),
					zap.Uint64("start_height", pollerBlocks[0].Height),
					zap.Uint64("end_height", targetHeight),
				)
				continue
			}
			fp.logger.Debug("the last publicRandCommit",
				zap.Uint64("rand_commit_start_height", lastPublicRandCommit.StartHeight),
				zap.Uint64("rand_commit_end_height", lastPublicRandCommit.EndHeight()),
				zap.Uint64("start_height", pollerBlocks[0].Height),
				zap.Uint64("end_height", targetHeight),
			)

			processedBlocks, err := fp.processBlocksToVote(pollerBlocks)
			if err != nil {
				fp.reportCriticalErr(err)

				continue
			}

			if len(processedBlocks) == 0 {
				continue
			}

			res, err := fp.retrySubmitSigsUntilFinalized(processedBlocks)
			if err != nil {
				fp.metrics.IncrementFpTotalFailedVotes(fp.GetBtcPkHex())
				if errors.Is(err, service.ErrFinalityProviderJailed) {
					fp.MustSetStatus(proto.FinalityProviderStatus_JAILED)
					fp.logger.Debug("the finality-provider has been jailed",
						zap.String("pk", fp.GetBtcPkHex()))

					continue
				}
				if !errors.Is(err, service.ErrFinalityProviderShutDown) {
					fp.reportCriticalErr(err)
				}

				continue
			}
			if res == nil {
				// this can happen when a finality signature is not needed
				// either if the block is already submitted or the signature
				// is already submitted
				continue
			}
			fp.logger.Info(
				"successfully submitted the finality signature to the consumer chain",
				zap.String("consumer_id", string(fp.GetChainID())),
				zap.String("pk", fp.GetBtcPkHex()),
				zap.Uint64("start_height", pollerBlocks[0].Height),
				zap.Uint64("end_height", targetHeight),
				zap.String("tx_hash", res.TxHash),
			)
		case <-fp.quit:
			fp.logger.Info("the finality signature submission loop is closing")

			return
		}
	}
}

// processBlocksToVote processes a batch a blocks and picks ones that need to vote
// it also updates the fp instance status according to the block's voting power
func (fp *FinalityProviderInstance) processBlocksToVote(blocks []*types.BlockInfo) ([]*types.BlockInfo, error) {
	processedBlocks := make([]*types.BlockInfo, 0, len(blocks))

	var hasPower bool
	for _, b := range blocks {
		blk := *b
		if blk.Height <= fp.GetLastVotedHeight() {
			fp.logger.Debug(
				"the block height is lower than last processed height",
				zap.String("pk", fp.GetBtcPkHex()),
				zap.Uint64("block_height", blk.Height),
				zap.Uint64("last_voted_height", fp.GetLastVotedHeight()),
			)

			continue
		}

		// check whether the finality provider has voting power
		hasPower, err := fp.GetVotingPowerWithRetry(blk.Height)
		if err != nil {
			return nil, fmt.Errorf("failed to get voting power for height %d: %w", blk.Height, err)
		}
		if !hasPower {
			fp.logger.Debug(
				"the finality-provider does not have voting power",
				zap.String("pk", fp.GetBtcPkHex()),
				zap.Uint64("block_height", blk.Height),
			)

			// the finality provider does not have voting power
			// and it will never will at this block, so continue
			fp.metrics.IncrementFpTotalBlocksWithoutVotingPower(fp.GetBtcPkHex())

			continue
		}

		processedBlocks = append(processedBlocks, &blk)
	}

	// update fp status according to the power for the last block
	if hasPower && fp.GetStatus() != proto.FinalityProviderStatus_ACTIVE {
		fp.MustSetStatus(proto.FinalityProviderStatus_ACTIVE)
	}

	if !hasPower && fp.GetStatus() == proto.FinalityProviderStatus_ACTIVE {
		fp.MustSetStatus(proto.FinalityProviderStatus_INACTIVE)
	}

	return processedBlocks, nil
}

func (fp *FinalityProviderInstance) getBatchBlocksFromChan() []*types.BlockInfo {
	var pollerBlocks []*types.BlockInfo
	for {
		select {
		case b := <-fp.poller.GetBlockInfoChan():
			pollerBlocks = append(pollerBlocks, b)
			if len(pollerBlocks) == int(fp.cfg.BatchSubmissionSize) {
				return pollerBlocks
			}
		case <-fp.quit:
			fp.logger.Info("the get all blocks loop is closing")

			return nil
		default:
			return pollerBlocks
		}
	}
}

func (fp *FinalityProviderInstance) randomnessCommitmentLoop() {
	defer fp.wg.Done()

	for {
		select {
		case <-time.After(fp.cfg.RandomnessCommitInterval):
			// start randomness commit in the first iteration
			should, startHeight, err := fp.ShouldCommitRandomness()
			if err != nil {
				fp.reportCriticalErr(err)

				continue
			}
			if !should {
				continue
			}

			txRes, err := fp.CommitPubRand(startHeight)
			if err != nil {
				fp.metrics.IncrementFpTotalFailedRandomness(fp.GetBtcPkHex())
				fp.reportCriticalErr(err)

				fp.logger.Sugar().Error("commit pub random failed", "err", err)

				continue
			}
			// txRes could be nil if no need to commit more randomness
			if txRes != nil {
				fp.logger.Info(
					"successfully committed public randomness to the consumer chain",
					zap.String("consumer_id", string(fp.GetChainID())),
					zap.String("pk", fp.GetBtcPkHex()),
					zap.String("tx_hash", txRes.TxHash),
				)
			}
		case <-fp.quit:
			fp.logger.Info("the randomness commitment loop is closing")

			return
		}
	}
}

// ShouldCommitRandomness determines whether a new randomness commit should be made
// Note: there's a delay from the commit is submitted to it is available to use due
// to timestamping. Therefore, the start height of the commit should consider an
// estimated delay.
// If randomness should be committed, start height of the commit will be returned
func (fp *FinalityProviderInstance) ShouldCommitRandomness() (bool, uint64, error) {
	return fp.inner.ShouldCommitRandomness()
}

func (fp *FinalityProviderInstance) reportCriticalErr(err error) {
	fp.criticalErrChan <- &CriticalError{
		err:     err,
		fpBtcPk: fp.GetBtcPkBIP340(),
	}
}

// retrySubmitSigsUntilFinalized periodically tries to submit finality signature until success or the block is finalized
// error will be returned if maximum retries have been reached or the query to the consumer chain fails
func (fp *FinalityProviderInstance) retrySubmitSigsUntilFinalized(targetBlocks []*types.BlockInfo) (*types.TxResponse, error) {
	if len(targetBlocks) == 0 {
		return nil, fmt.Errorf("cannot send signatures for empty blocks")
	}

	var failedCycles uint32
	targetHeight := targetBlocks[len(targetBlocks)-1].Height

	// First iteration happens before the loop
	for {
		// Attempt submission immediately
		// error will be returned if max retries have been reached
		var res *types.TxResponse
		var err error

		for _, target := range targetBlocks {
			if target.Height != 0 {
				fp.blitzMetrics.RecordCommittedHeight(fp.GetBtcPkHex(), target.Height)
			}
		}
		fp.recordOrbitFinalizedHeight()

		res, err = fp.SubmitBatchFinalitySignatures(targetBlocks)
		if err != nil {
			fp.logger.Debug(
				"failed to submit finality signature to the consumer chain",
				zap.String("pk", fp.GetBtcPkHex()),
				zap.Uint32("current_failures", failedCycles),
				zap.Uint64("target_start_height", targetBlocks[0].Height),
				zap.Uint64("target_end_height", targetHeight),
				zap.Error(err),
			)

			if fpcc.IsUnrecoverable(err) {
				return nil, err
			}

			if fpcc.IsExpected(err) {
				return nil, nil
			}

			failedCycles++
			if failedCycles > fp.cfg.MaxSubmissionRetries {
				return nil, fmt.Errorf("reached max failed cycles with err: %w", err)
			}
		} else {
			for _, target := range targetBlocks {
				if target.Height != 0 {
					fp.blitzMetrics.RecordOrbitBabylonFinalizedHeight(fp.GetBtcPkHex(), target.Height, target.Hash)
				}
			}
			fp.recordOrbitFinalizedHeight()

			// The signature has been successfully submitted
			return res, nil
		}

		// periodically query the index block to be later checked whether it is Finalized
		finalized, err := fp.checkBlockFinalization(targetHeight)
		if err != nil {
			return nil, fmt.Errorf("failed to query block finalization at height %v: %w", targetHeight, err)
		}
		if finalized {
			fp.logger.Debug(
				"the block is already finalized, skip submission",
				zap.String("pk", fp.GetBtcPkHex()),
				zap.Uint64("target_height", targetHeight),
			)

			fp.metrics.IncrementFpTotalFailedVotes(fp.GetBtcPkHex())

			// TODO: returning nil here is to safely break the loop
			//  the error still exists
			return nil, nil
		}

		// Wait for the retry interval
		select {
		case <-time.After(fp.cfg.SubmissionRetryInterval):
			// Continue to next retry iteration
			continue
		case <-fp.quit:
			fp.logger.Debug("the finality-provider instance is closing", zap.String("pk", fp.GetBtcPkHex()))

			return nil, service.ErrFinalityProviderShutDown
		}
	}
}

func (fp *FinalityProviderInstance) checkBlockFinalization(height uint64) (bool, error) {
	b, err := fp.consumerCon.QueryBlock(height)
	if err != nil {
		return false, err
	}

	return b.Finalized, nil
}

// CommitPubRand commits a list of randomness from given start height
func (fp *FinalityProviderInstance) CommitPubRand(startHeight uint64) (*types.TxResponse, error) {
	fp.logger.Sugar().Debug("CommitPubRand", "start", startHeight)

	return fp.inner.CommitPubRand(startHeight)
}

func (fp *FinalityProviderInstance) TestCommitPubRand(targetBlockHeight uint64) error {
	return fp.TestCommitPubRand(targetBlockHeight)
}

func (fp *FinalityProviderInstance) TestCommitPubRandWithStartHeight(startHeight uint64, targetBlockHeight uint64) error {
	return fp.TestCommitPubRandWithStartHeight(startHeight, targetBlockHeight)
}

// SubmitFinalitySignature builds and sends a finality signature over the given block to the consumer chain
func (fp *FinalityProviderInstance) SubmitFinalitySignature(b *types.BlockInfo) (*types.TxResponse, error) {
	return fp.inner.SubmitFinalitySignature(b)
}

// SubmitBatchFinalitySignatures builds and sends a finality signature over the given block to the consumer chain
// NOTE: the input blocks should be in the ascending order of height
func (fp *FinalityProviderInstance) SubmitBatchFinalitySignatures(blocks []*types.BlockInfo) (*types.TxResponse, error) {
	return fp.inner.SubmitBatchFinalitySignatures(blocks)
}

// TestSubmitFinalitySignatureAndExtractPrivKey is exposed for presentation/testing purpose to allow manual sending finality signature
// this API is the same as SubmitBatchFinalitySignatures except that we don't constraint the voting height and update status
// Note: this should not be used in the submission loop
func (fp *FinalityProviderInstance) TestSubmitFinalitySignatureAndExtractPrivKey(
	b *types.BlockInfo, useSafeEOTSFunc bool,
) (*types.TxResponse, *btcec.PrivateKey, error) {
	return fp.TestSubmitFinalitySignatureAndExtractPrivKey(b, useSafeEOTSFunc)
}

// DetermineStartHeight determines start height for block processing by:
//
// If AutoChainScanningMode is disabled:
//   - Returns StaticChainScanningStartHeight from config
//
// If AutoChainScanningMode is enabled:
//   - Gets finalityActivationHeight from chain
//   - Gets lastFinalizedHeight from chain
//   - Gets lastVotedHeight from local state
//   - Gets highestVotedHeight from chain
//   - Sets startHeight = max(lastVotedHeight, highestVotedHeight, lastFinalizedHeight) + 1
//   - Returns max(startHeight, finalityActivationHeight) to ensure startHeight is not
//     lower than the finality activation height
//
// This ensures that:
// 1. The FP will not vote for heights below the finality activation height
// 2. The FP will resume from its last voting position or the chain's last finalized height
// 3. The FP will not process blocks it has already voted on
//
// Note: Starting from lastFinalizedHeight when there's a gap to the last processed height
// may result in missed rewards, depending on the consumer chain's reward distribution mechanism.
func (fp *FinalityProviderInstance) DetermineStartHeight() (uint64, error) {
	return fp.inner.DetermineStartHeight()
}

func (fp *FinalityProviderInstance) GetLastCommittedHeight() (uint64, error) {
	return fp.inner.GetLastCommittedHeight()
}

func (fp *FinalityProviderInstance) GetVotingPowerWithRetry(height uint64) (bool, error) {
	return fp.inner.GetVotingPowerWithRetry(height)
}

func (fp *FinalityProviderInstance) GetFinalityProviderSlashedOrJailedWithRetry() (bool, bool, error) {
	return fp.inner.GetFinalityProviderSlashedOrJailedWithRetry()
}
