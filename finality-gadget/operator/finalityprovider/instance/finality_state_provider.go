package fp_instance

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"

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
	"github.com/pkg/errors"
)

const FastCheckNumberCount uint64 = 256

type FinalizedStateProvider struct {
	logger    *zap.Logger
	l2Client  *l2eth.L2EthClient
	btcClient finalitygadget.IBitcoinClient
	bbnClient finalitygadget.IBabylonClient
	cwClient  finalitygadget.ICosmWasmClient

	lastFinalizedHeight uint64
	mu                  sync.Mutex
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
	cwClient := cwclient.NewCosmWasmClient(babylonClient.QueryClient.RPCClient, cfg.Babylon.FinalityGadgetCfg.FGContractAddress)

	l2Client, err := l2eth.NewL2EthClient(ctx, &cfg.Layer2)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create l2 eth client")
	}

	return &FinalizedStateProvider{
		logger:    logger,
		l2Client:  l2Client,
		btcClient: btcClient,
		bbnClient: bbnClient,
		cwClient:  cwClient,
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

	if currentNumber > FastCheckNumberCount {
		checkNumber := currentNumber - FastCheckNumberCount
		isFinalized, err := p.queryFinalizedBlockInBabylonByNumber(ctx, checkNumber)
		if err != nil {
			return 0, errors.Wrapf(err, "queryFinalizedBlockInBabylonByNumber failed: %v", checkNumber)
		}

		if isFinalized {
			fromBlockHeight = checkNumber
		}
	}

	res, err := p.queryFinalizedBlockInBabylonFromTo(ctx, fromBlockHeight, currentNumber)
	if err == nil {
		p.SetLastFinalized(res)
	}

	return res, err
}

func (p *FinalizedStateProvider) queryFinalizedBlockInBabylonByNumber(ctx context.Context, height uint64) (bool, error) {
	blk, err := p.l2Client.BlockByNumber(ctx, big.NewInt(int64(height)))
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

	check := (from + to + 1) / 2

	p.logger.Sugar().Debugf("queryFinalizedBlockInBabylonFromTo from %v to %v check %v", from, to, check)

	isFinalized, err := p.queryFinalizedBlockInBabylonByNumber(ctx, check)
	if err != nil {
		return 0, errors.Wrapf(err, "queryFinalizedBlockInBabylonByNumber failed: %v", check)
	}

	p.logger.Sugar().Debugf("queryFinalizedBlockInBabylonByNumber got isFinalized: %v", isFinalized)
	if isFinalized {
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

	// get all FPs pubkey for the consumer chain
	allFpPks, err := p.queryAllFpBtcPubKeys()
	if err != nil {
		return false, errors.Wrap(err, "queryAllFpBtcPubKeys")
	}

	// convert the L2 timestamp to BTC height
	btcblockHeight, err := p.btcClient.GetBlockHeightByTimestamp(block.BlockTimestamp)
	if err != nil {
		return false, errors.Wrap(err, "GetBlockHeightByTimestamp")
	}

	// check whether the btc staking is actived
	earliestDelHeight, err := p.bbnClient.QueryEarliestActiveDelBtcHeight(allFpPks)
	if err != nil {
		return false, errors.Wrap(err, "QueryEarliestActiveDelBtcHeight")
	}
	if btcblockHeight < earliestDelHeight {
		return false, types.ErrBtcStakingNotActivated
	}

	// get all FPs voting power at this BTC height
	allFpPower, err := p.bbnClient.QueryMultiFpPower(allFpPks, btcblockHeight)
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
		return false, types.ErrNoFpHasVotingPower
	}

	// get all FPs that voted this (L2 block height, L2 block hash) combination
	votedFpPks, err := p.cwClient.QueryListOfVotedFinalityProviders(block)
	if err != nil {
		return false, errors.Wrap(err, "QueryListOfVotedFinalityProviders")
	}
	if votedFpPks == nil {
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
		return false, nil
	}
	return true, nil
}

func (p *FinalizedStateProvider) queryAllFpBtcPubKeys() ([]string, error) {
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
	return allFpPks, nil
}
