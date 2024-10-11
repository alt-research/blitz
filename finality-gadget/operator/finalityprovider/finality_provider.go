package finalityprovider

import (
	"context"
	"sync"
	"time"

	bbnclient "github.com/babylonlabs-io/babylon/client/client"
	bbncfg "github.com/babylonlabs-io/babylon/client/config"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/operator/finalityprovider/cwclient"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type FinalityProvider struct {
	logger logging.Logger
	cfg    *Config
	BtcPk  *btcec.PublicKey

	cwClient     cwclient.ICosmosWasmContractClient
	tickInterval time.Duration

	l2BlocksChan chan *types.Block

	wg sync.WaitGroup
}

var _ IFinalityProvider = &FinalityProvider{}

func NewFinalityProvider(
	ctx context.Context,
	cfg *Config,
	logger logging.Logger,
	zaplogger *zap.Logger) (*FinalityProvider, error) {
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

	cwClient := cwclient.NewCosmWasmClient(
		logger.With("module", "cosmWasmClient"),
		babylonClient.QueryClient.RPCClient,
		btcPk,
		cfg.BtcPk,
		cfg.FgContractAddress)

	return &FinalityProvider{
		logger:       logger,
		cfg:          cfg,
		BtcPk:        btcPk,
		cwClient:     cwClient,
		tickInterval: 1 * time.Second,
	}, nil
}
