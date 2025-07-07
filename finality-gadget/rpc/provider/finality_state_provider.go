package provider

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	bbnclient "github.com/babylonlabs-io/babylon/client/client"
	bbncfg "github.com/babylonlabs-io/babylon/client/config"
	fgbbnclient "github.com/babylonlabs-io/finality-gadget/bbnclient"
	"github.com/babylonlabs-io/finality-gadget/btcclient"
	"github.com/babylonlabs-io/finality-gadget/cwclient"
	"github.com/babylonlabs-io/finality-gadget/finalitygadget"
	"github.com/babylonlabs-io/finality-gadget/testutil/mocks"
	"github.com/babylonlabs-io/finality-gadget/types"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

const FastCheckNumberCount uint64 = 256
const CacheMapCount int = 4096

type FinalizedStateProvider struct {
	logger    *zap.Logger
	l2Client  *l2eth.L2EthClient
	btcClient finalitygadget.IBitcoinClient
	bbnClient finalitygadget.IBabylonClient
	cwClient  finalitygadget.ICosmWasmClient

	lastFinalizedHeight uint64
	mu                  sync.Mutex

	allFpsCache                     []string
	allFpsCacheLastTime             time.Time
	votedFpPksCache                 map[string][]string
	finalizedCache                  map[uint64]bool
	btcblockHeightCache             map[string]uint32
	earliestActiveDelBtcHeightCache map[string]uint32
	multiFpPowerCache               map[uint32]map[string]uint64
	l2BlockCache                    map[uint64]*ethTypes.Block
	cacheMu                         sync.RWMutex
}

func NewFinalizedStateProvider(
	ctx context.Context,
	cfg *configs.OperatorConfig,
	logger *zap.Logger) (*FinalizedStateProvider, error) {
	// Create babylon client
	bbnConfig := bbncfg.DefaultBabylonConfig()
	bbnConfig.RPCAddr = cfg.Babylon.FinalityGadgetCfg.BBNRPCAddress
	bbnConfig.ChainID = cfg.Babylon.FinalityGadgetCfg.BBNChainID
	babylonClient, err := bbnclient.New(
		&bbnConfig,
		logger,
	)
	bbnClient := fgbbnclient.NewBabylonClient(babylonClient.QueryClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create Babylon client: %w", err)
	}

	// Create bitcoin client
	btcConfig := btcclient.DefaultBTCConfig()
	btcConfig.RPCHost = cfg.Babylon.FinalityGadgetCfg.BitcoinRPCHost
	if cfg.Babylon.FinalityGadgetCfg.BitcoinRPCUser != "" && cfg.Babylon.FinalityGadgetCfg.BitcoinRPCPass != "" {
		btcConfig.RPCUser = cfg.Babylon.FinalityGadgetCfg.BitcoinRPCUser
		btcConfig.RPCPass = cfg.Babylon.FinalityGadgetCfg.BitcoinRPCPass
	}
	if cfg.Babylon.FinalityGadgetCfg.BitcoinDisableTLS {
		btcConfig.DisableTLS = true
	}
	var btcClient finalitygadget.IBitcoinClient
	switch cfg.Babylon.FinalityGadgetCfg.BitcoinRPCHost {
	case "mock-btc-client":
		btcClient, err = mocks.NewMockBitcoinClient(btcConfig, logger)
	default:
		btcClient, err = btcclient.NewBitcoinClient(btcConfig, logger)
	}
	if err != nil {
		return nil, err
	}

	// Create cosmwasm client
	logger.Sugar().Infof("the fg contract address %s", cfg.Babylon.FinalityGadgetCfg.FGContractAddress)
	cwClient := cwclient.NewCosmWasmClient(babylonClient.QueryClient.RPCClient, cfg.Babylon.FinalityGadgetCfg.FGContractAddress)

	l2Client, err := l2eth.NewL2EthClient(ctx, &cfg.Layer2)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create l2 eth client")
	}

	return &FinalizedStateProvider{
		logger:                          logger,
		l2Client:                        l2Client,
		btcClient:                       btcClient,
		bbnClient:                       bbnClient,
		cwClient:                        cwClient,
		votedFpPksCache:                 make(map[string][]string, CacheMapCount),
		finalizedCache:                  make(map[uint64]bool, CacheMapCount),
		btcblockHeightCache:             make(map[string]uint32, CacheMapCount),
		earliestActiveDelBtcHeightCache: make(map[string]uint32, CacheMapCount),
		multiFpPowerCache:               make(map[uint32]map[string]uint64, CacheMapCount),
		l2BlockCache:                    make(map[uint64]*ethTypes.Block, CacheMapCount),
	}, nil
}

func (p *FinalizedStateProvider) GetLastFinalized() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.lastFinalizedHeight
}

func (p *FinalizedStateProvider) SetLastFinalized(height uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.lastFinalizedHeight < height {
		p.lastFinalizedHeight = height
	}
}

func (p *FinalizedStateProvider) QueryFinalizedBlockInBabylon(ctx context.Context) (uint64, error) {
	currentNumber, err := p.l2Client.BlockNumber(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "failed to got blockNumber")
	}

	fromBlockHeight := p.GetLastFinalized()
	if fromBlockHeight == 0 {
		fromBlockHeight = 1
	}

	if currentNumber == fromBlockHeight {
		return currentNumber, nil
	}

	// from block height is the start search point
	// mostly if there is no new block, we can just return it

	// check current newest block
	if currentNumber > 1 {
		isFinalized, err := p.queryFinalizedBlockInBabylonByNumber(ctx, currentNumber-1)
		if err == nil {
			if isFinalized {
				// The finalitzed block number can be the current number -1
				return currentNumber - 1, nil
			}
		}
	}

	// if last finalized block is too earlier, we can just use 256 block to try
	if currentNumber > FastCheckNumberCount {
		checkNumber := currentNumber - FastCheckNumberCount
		if checkNumber > fromBlockHeight {
			p.logger.Sugar().Debugf(
				"try use a check number near the current header: %d, %d, %d",
				fromBlockHeight, checkNumber, currentNumber)
			isFinalized, err := p.queryFinalizedBlockInBabylonByNumber(ctx, checkNumber)
			if err != nil {
				return 0, errors.Wrapf(err, "queryFinalizedBlockInBabylonByNumber failed: %v", checkNumber)
			}

			if isFinalized {
				fromBlockHeight = checkNumber
			}
		}
	}

	// mostly the currentNumber is the next block to last finality, so we can check it fast
	nextFinality := fromBlockHeight + 1
	if nextFinality == currentNumber {
		p.logger.Sugar().Debugf(
			"just check the current header by it is the next: %d, %d",
			fromBlockHeight, currentNumber)
		isFinalized, err := p.queryFinalizedBlockInBabylonByNumber(ctx, currentNumber)
		if err != nil {
			return 0, errors.Wrapf(err, "queryFinalizedBlockInBabylonByNumber failed: %v", currentNumber)
		}

		if isFinalized {
			return currentNumber, nil
		} else {
			return fromBlockHeight, nil
		}
	}

	// mostly it is no new finality block after last finality, so we can check next
	tryEndBlockHeight := fromBlockHeight + 1
	if tryEndBlockHeight < currentNumber {
		p.logger.Sugar().Debugf("try use next finaliy block to check: %d, %d", tryEndBlockHeight, currentNumber)

		isFinalized, err := p.queryFinalizedBlockInBabylonByNumber(ctx, tryEndBlockHeight)
		if err != nil {
			return 0, errors.Wrapf(err, "queryFinalizedBlockInBabylonByNumber failed: %v", tryEndBlockHeight)
		}

		if isFinalized {
			fromBlockHeight = tryEndBlockHeight
		} else {
			currentNumber = tryEndBlockHeight
		}
	}

	res, err := p.queryFinalizedBlockInBabylonFromTo(ctx, fromBlockHeight, currentNumber)
	if err == nil {
		p.SetLastFinalized(res)
	}

	return res, err
}

func (p *FinalizedStateProvider) blockByNumber(ctx context.Context, number uint64) (*ethTypes.Block, error) {
	res, useCache := func() (*ethTypes.Block, bool) {
		p.cacheMu.RLock()
		defer p.cacheMu.RUnlock()

		c, ok := p.l2BlockCache[number]
		if ok {
			return c, true
		}

		return nil, false
	}()

	if useCache {
		return res, nil
	}

	blk, err := p.l2Client.BlockByNumber(ctx, big.NewInt(int64(number)))
	if err != nil {
		return nil, errors.Wrapf(err, "QueryBlock failed: %v", number)
	}

	if err == nil {
		func() {
			p.cacheMu.Lock()
			defer p.cacheMu.Unlock()

			if len(p.l2BlockCache) > CacheMapCount {
				p.l2BlockCache = make(map[uint64]*ethTypes.Block, CacheMapCount)
			}

			p.l2BlockCache[number] = blk
		}()
	}

	return blk, err
}

func (p *FinalizedStateProvider) queryFinalizedBlockInBabylonByNumber(ctx context.Context, height uint64) (bool, error) {
	useCache := func() bool {
		p.cacheMu.RLock()
		defer p.cacheMu.RUnlock()

		_, ok := p.finalizedCache[height]

		return ok
	}()

	if useCache {
		p.logger.Sugar().Debugf("queryFinalizedBlockInBabylonByNumber final by cache: %d", height)
		return true, nil
	}

	blk, err := p.blockByNumber(ctx, height)
	if err != nil {
		return false, errors.Wrapf(err, "QueryBlock failed: %v", height)
	}

	isFinalized, err := p.QueryIsBlockBabylonFinalizedFromBabylon(&types.Block{
		BlockHash:      blk.Hash().Hex(),
		BlockTimestamp: blk.Time(),
		BlockHeight:    blk.NumberU64(),
	})
	if err != nil {
		return false, errors.Wrapf(err, "QueryIsBlockBabylonFinalizedFromBabylon failed: %v", height)
	}

	if isFinalized {
		func() {
			p.cacheMu.Lock()
			defer p.cacheMu.Unlock()

			if len(p.finalizedCache) > CacheMapCount {
				p.logger.Sugar().Debugf("clean the finality cache to %d", CacheMapCount)
				p.finalizedCache = make(map[uint64]bool, CacheMapCount)
			}

			p.logger.Sugar().Debugf("fill into the new finality cache %d", height)

			p.finalizedCache[height] = true
		}()
	}

	p.logger.Sugar().Debugf("queryFinalizedBlockInBabylonByNumber: %d, %v", height, isFinalized)

	return isFinalized, nil
}

func (p *FinalizedStateProvider) queryFinalizedBlockInBabylonFromTo(ctx context.Context, from, to uint64) (uint64, error) {
	// from is a finalized block
	if from > to {
		tmp := from
		from = to
		to = tmp
	}

	if from == to {
		return to, nil
	}

	if from+1 == to {
		return from, nil
	}

	check := (from + to + 1) / 2

	p.logger.Sugar().Debugf("queryFinalizedBlockInBabylonFromTo from %v to %v check %v", from, to, check)

	isFinalized, err := p.queryFinalizedBlockInBabylonByNumber(ctx, check)
	if err != nil {
		return 0, errors.Wrapf(err, "queryFinalizedBlockInBabylonByNumber failed: %v", check)
	}

	p.logger.Sugar().Debugf("queryFinalizedBlockInBabylonByNumber got isFinalized: %v", isFinalized)
	if isFinalized {
		p.SetLastFinalized(check - 1)
		return p.queryFinalizedBlockInBabylonFromTo(ctx, check, to)
	} else {
		return p.queryFinalizedBlockInBabylonFromTo(ctx, from, check)
	}
}

// TODO: make this method internal once fully tested. External services should query the database instead.
/* QueryIsBlockBabylonFinalizedFromBabylon checks if the given L2 block is finalized by querying the Babylon node
 *
 * - if the finality gadget is not enabled, always return true
 * - else, check if the given L2 block is finalized
 * - return true if finalized, false if not finalized, and error if any
 *
 * - to check if the block is finalized, we need to:
 *   - get the consumer chain id
 *   - get all the FPs pubkey for the consumer chain
 *   - convert the L2 block timestamp to BTC height
 *   - get all FPs voting power at this BTC height
 *   - calculate total voting power
 *   - get all FPs that voted this L2 block with the same height and hash
 *   - calculate voted voting power
 *   - check if the voted voting power is more than 2/3 of the total voting power
 */
func (p *FinalizedStateProvider) QueryIsBlockBabylonFinalizedFromBabylon(block *types.Block) (bool, error) {
	if block == nil {
		return false, fmt.Errorf("block is nil")
	}

	// trim prefix 0x for the L2 block hash
	block.BlockHash = strings.TrimPrefix(block.BlockHash, "0x")

	// get all FPs voting power at this BTC height
	allFpPower, err := p.queryAllPkPower(block)
	if err != nil {
		return false, errors.Wrap(err, "QueryMultiFpPower")
	}

	// calculate total voting power
	var totalPower uint64 = 0
	for _, power := range allFpPower {
		totalPower += power
	}

	// no FP has voting power for the consumer chain
	if totalPower == 0 {
		p.logger.Sugar().Debugf("block not finalized by no totalPower for %v", block.BlockHeight)
		return true, nil
	}

	// get all FPs that voted this (L2 block height, L2 block hash) combination
	votedFpPks, err := p.queryListOfVotedFinalityProviders(block)
	if err != nil {
		return false, errors.Wrap(err, "QueryListOfVotedFinalityProviders")
	}
	if votedFpPks == nil {
		p.logger.Sugar().Debugw("votedFpPks nil", "height", block.BlockHeight)
		return false, nil
	}
	// calculate voted voting power
	var votedPower uint64 = 0
	for _, key := range votedFpPks {
		if power, exists := allFpPower[key]; exists {
			votedPower += power
		}
	}

	// quorom < 2/3
	if votedPower*3 < totalPower*2 {
		p.logger.Sugar().Debugf("voted power no enough %v to %v", votedPower, totalPower)
		return false, nil
	}
	return true, nil
}

func (p *FinalizedStateProvider) queryAllFpBtcPubKeys() ([]string, error) {
	res, useCache := func() ([]string, bool) {
		p.cacheMu.RLock()
		defer p.cacheMu.RUnlock()

		if len(p.allFpsCache) > 0 && isTimeGreaterThan(p.allFpsCacheLastTime) {
			return p.allFpsCache, true
		}

		return nil, false
	}()

	if useCache {
		return res, nil
	}

	// get the consumer chain id
	consumerId, err := p.cwClient.QueryConsumerId()
	if err != nil {
		return nil, err
	}

	// get all the FPs pubkey for the consumer chain
	allFpPks, err := p.bbnClient.QueryAllFpBtcPubKeys(consumerId)
	if err != nil {
		return nil, err
	}

	func() {
		p.cacheMu.Lock()
		defer p.cacheMu.Unlock()

		p.allFpsCache = allFpPks
		p.allFpsCacheLastTime = time.Now()
	}()

	return allFpPks, nil
}

func (p *FinalizedStateProvider) queryListOfVotedFinalityProviders(queryParams *types.Block) ([]string, error) {
	res, useCache := func() ([]string, bool) {
		p.cacheMu.RLock()
		defer p.cacheMu.RUnlock()

		c, ok := p.votedFpPksCache[queryParams.BlockHash]
		if ok {
			return c, true
		}

		return nil, false
	}()

	if useCache {
		return res, nil
	}

	votedFpPks, err := p.cwClient.QueryListOfVotedFinalityProviders(queryParams)

	if err == nil {
		func() {
			p.cacheMu.Lock()
			defer p.cacheMu.Unlock()

			if len(p.votedFpPksCache) > CacheMapCount {
				p.votedFpPksCache = make(map[string][]string, CacheMapCount)
			}

			p.votedFpPksCache[queryParams.BlockHash] = votedFpPks
		}()
	}

	if len(votedFpPks) == 0 {
		p.logger.Sugar().Debugw("not found voted finality provider", "block", queryParams.BlockHeight)
	}

	return votedFpPks, err
}

func (p *FinalizedStateProvider) getBlockHeightByTimestamp(block *types.Block) (uint32, error) {
	res, useCache := func() (uint32, bool) {
		p.cacheMu.RLock()
		defer p.cacheMu.RUnlock()

		c, ok := p.btcblockHeightCache[block.BlockHash]
		if ok {
			return c, true
		}

		return 0, false
	}()

	if useCache {
		return res, nil
	}

	// convert the L2 timestamp to BTC height
	btcblockHeight, err := p.btcClient.GetBlockHeightByTimestamp(block.BlockTimestamp)
	if err != nil {
		return 0, errors.Wrap(err, "GetBlockHeightByTimestamp")
	}

	if err == nil {
		func() {
			p.cacheMu.Lock()
			defer p.cacheMu.Unlock()

			if len(p.btcblockHeightCache) > CacheMapCount {
				p.btcblockHeightCache = make(map[string]uint32, CacheMapCount)
			}

			p.btcblockHeightCache[block.BlockHash] = btcblockHeight
		}()
	}

	return btcblockHeight, err

}

func (p *FinalizedStateProvider) queryEarliestActiveDelBtcHeight(fpPubkeyHexList []string) (uint32, error) {
	key := strings.Join(fpPubkeyHexList, ",")

	res, useCache := func() (uint32, bool) {
		p.cacheMu.RLock()
		defer p.cacheMu.RUnlock()

		c, ok := p.earliestActiveDelBtcHeightCache[key]
		if ok {
			return c, true
		}

		return 0, false
	}()

	if useCache {
		return res, nil
	}

	// check whether the btc staking is actived
	earliestDelHeight, err := p.bbnClient.QueryEarliestActiveDelBtcHeight(fpPubkeyHexList)
	if err != nil {
		return 0, errors.Wrap(err, "QueryEarliestActiveDelBtcHeight")
	}

	if err == nil {
		func() {
			p.cacheMu.Lock()
			defer p.cacheMu.Unlock()

			if len(p.earliestActiveDelBtcHeightCache) > CacheMapCount {
				p.earliestActiveDelBtcHeightCache = make(map[string]uint32, CacheMapCount)
			}

			p.earliestActiveDelBtcHeightCache[key] = earliestDelHeight
		}()
	}

	return earliestDelHeight, err
}

func (p *FinalizedStateProvider) queryMultiFpPower(fpPubkeyHexList []string, btcHeight uint32) (map[string]uint64, error) {
	res, useCache := func() (map[string]uint64, bool) {
		p.cacheMu.RLock()
		defer p.cacheMu.RUnlock()

		c, ok := p.multiFpPowerCache[btcHeight]
		if ok {
			return c, true
		}

		return nil, false
	}()

	if useCache {
		return res, nil
	}

	// get all FPs voting power at this BTC height
	allFpPower, err := p.bbnClient.QueryMultiFpPower(fpPubkeyHexList, btcHeight)
	if err != nil {
		return nil, errors.Wrap(err, "QueryMultiFpPower")
	}

	if err == nil {
		func() {
			p.cacheMu.Lock()
			defer p.cacheMu.Unlock()

			if len(p.multiFpPowerCache) > CacheMapCount {
				p.multiFpPowerCache = make(map[uint32]map[string]uint64, CacheMapCount)
			}

			p.multiFpPowerCache[btcHeight] = allFpPower
		}()
	}

	return allFpPower, err
}

func (p *FinalizedStateProvider) queryAllPkPower(block *types.Block) (map[string]uint64, error) {
	// get all FPs pubkey for the consumer chain
	allFpPks, err := p.queryAllFpBtcPubKeys()
	if err != nil {
		return nil, errors.Wrap(err, "queryAllFpBtcPubKeys")
	}

	p.logger.Sugar().Infof("allFpPks %v", allFpPks)

	// convert the L2 timestamp to BTC height
	btcblockHeight, err := p.getBlockHeightByTimestamp(block)
	if err != nil {
		return nil, errors.Wrap(err, "GetBlockHeightByTimestamp")
	}

	p.logger.Sugar().Infof("btcblockHeight %v", btcblockHeight)

	// check whether the btc staking is actived
	earliestDelHeight, err := p.queryEarliestActiveDelBtcHeight(allFpPks)
	if err != nil {
		return nil, errors.Wrap(err, "QueryEarliestActiveDelBtcHeight")
	}

	p.logger.Sugar().Info("earliestDelHeight ", earliestDelHeight)

	if btcblockHeight < earliestDelHeight {
		//return nil, errors.Wrapf(types.ErrBtcStakingNotActivated, "current %v, earliest %v", btcblockHeight, earliestDelHeight)
	}

	// get all FPs voting power at this BTC height
	allFpPower, err := p.queryMultiFpPower(allFpPks, btcblockHeight)
	if err != nil {
		return nil, errors.Wrap(err, "QueryMultiFpPower")
	}

	p.logger.Sugar().Info("allFpPower ", allFpPower)

	return allFpPower, nil
}

func isTimeGreaterThan(inputTime time.Time) bool {
	if inputTime.IsZero() {
		return false
	}

	currentTime := time.Now()

	duration := currentTime.Sub(inputTime)

	return duration > 240*time.Second
}
