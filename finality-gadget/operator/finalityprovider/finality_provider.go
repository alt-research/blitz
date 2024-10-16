package finalityprovider

import (
	"context"
	"sync"
	"time"

	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
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

var _ IFinalityProvider = &FinalityProvider{}

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

	provider, err := cfg.ToCosmosProviderConfig().NewProvider(
		zaplogger,
		"", // TODO: set home path
		true,
		cfg.BbnChainID,
	)
	if err != nil {
		return nil, err
	}

	cp := provider.(*cosmos.CosmosProvider)
	cp.PCfg.KeyDirectory = cfg.Cosmwasm.KeyDirectory

	// initialise Cosmos provider
	// NOTE: this will create a RPC client. The RPC client will be used for
	// submitting txs and making ad hoc queries. It won't create WebSocket
	// connection with wasmd node
	err = cp.Init(ctx)
	if err != nil {
		return nil, err
	}

	cwClient := cwclient.NewCosmWasmClient(
		logger.With("module", "cosmWasmClient"),
		babylonClient.QueryClient.RPCClient,
		btcPk,
		cfg.BtcPk,
		cfg.FgContractAddress,
		cfg.FpAddr,
		cp)

	committer := NewL2BlockCommitter(logger, finalityGadgetClient, activatedHeight)

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
