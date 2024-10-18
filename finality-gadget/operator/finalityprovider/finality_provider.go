package finalityprovider

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	bbnclient "github.com/babylonlabs-io/babylon/client/client"
	bbncfg "github.com/babylonlabs-io/babylon/client/config"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/operator/finalityprovider/cwclient"
	sdkClient "github.com/alt-research/blitz/finality-gadget/sdk/client"
)

type FinalityProvider struct {
	logger logging.Logger
	cfg    *Config
	BtcPk  *btcec.PublicKey

	cwClient             cwclient.ICosmosWasmContractClient
	finalityGadgetClient sdkClient.IFinalityGadget
	committer            *L2BlockCommitter
	tickInterval         time.Duration

	l2BlocksChan chan *types.Block

	wg sync.WaitGroup
}

func NewFinalityProvider(
	ctx context.Context,
	cfg *Config,
	logger logging.Logger,
	zaplogger *zap.Logger,
	finalityGadgetClient sdkClient.IFinalityGadget,
	activatedHeight uint64,
) (*FinalityProvider, error) {
	btcPk, err := cfg.GetBtcPk()
	if err != nil {
		return nil, errors.Wrap(err, "get btc pk failed")
	}

	// Create babylon client
	bbnConfig := bbncfg.DefaultBabylonConfig()
	bbnConfig.RPCAddr = cfg.BbnRpcAddress
	bbnConfig.ChainID = cfg.BbnChainID
	babylonClient, err := bbnclient.New(
		&bbnConfig,
		zaplogger,
	)

	cp, err := NewProvider(ctx, cfg, zaplogger)
	if err != nil {
		return nil, errors.Wrap(err, "new provider failed")
	}

	key, err := cp.GetKeyAddressForKey(cfg.Cosmwasm.Key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get key address for %v", cfg.Cosmwasm.Key)
	}

	zaplogger.Sugar().Debug("key address", "name", cfg.Cosmwasm.Key, "address", key)

	cwClient := cwclient.NewCosmWasmClient(
		logger.With("module", "cosmWasmClient"),
		babylonClient.QueryClient.RPCClient,
		btcPk,
		cfg.BtcPk,
		cfg.FgContractAddress,
		cfg.FpAddr,
		cp)

	committer := NewL2BlockCommitter(
		logger,
		cfg,
		finalityGadgetClient,
		cwClient,
		activatedHeight,
		btcPk)

	return &FinalityProvider{
		logger:               logger,
		cfg:                  cfg,
		BtcPk:                btcPk,
		cwClient:             cwClient,
		committer:            committer,
		finalityGadgetClient: finalityGadgetClient,
		l2BlocksChan:         make(chan *types.Block, 32),
		tickInterval:         1 * time.Second,
	}, nil
}

func (p *FinalityProvider) OnBlock(ctx context.Context, blk *types.Block) error {
	p.l2BlocksChan <- blk

	return nil
}
