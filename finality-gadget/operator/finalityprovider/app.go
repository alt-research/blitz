package finalityprovider

import (
	"context"
	"fmt"
	"sync"

	"github.com/lightningnetwork/lnd/kvdb"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	bbntypes "github.com/babylonlabs-io/babylon/types"
	fpcc "github.com/babylonlabs-io/finality-provider/clientcontroller"
	ccapi "github.com/babylonlabs-io/finality-provider/clientcontroller/api"
	fpeotsmanager "github.com/babylonlabs-io/finality-provider/eotsmanager"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/babylonlabs-io/finality-provider/finality-provider/proto"
	"github.com/babylonlabs-io/finality-provider/finality-provider/store"
	fp_metrics "github.com/babylonlabs-io/finality-provider/metrics"

	"github.com/alt-research/blitz/finality-gadget/client/eotsmanager"
	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/metrics"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	"github.com/alt-research/blitz/finality-gadget/operator/finalityprovider/controllers"
	"github.com/alt-research/blitz/finality-gadget/rpc"
)

type FinalityProviderApp struct {
	startOnce sync.Once
	stopOnce  sync.Once

	quit chan struct{}

	config *fpcfg.Config

	fpManager   *FinalityProviderManager
	eotsManager fpeotsmanager.EOTSManager
	rpc         *rpc.JsonRpcServer
	logger      *zap.Logger

	jsonRpcServerIpPortAddr string

	wg sync.WaitGroup
}

func NewFinalityProviderAppFromConfig(
	ctx context.Context,
	cfg *configs.OperatorConfig,
	fpConfig *fpcfg.Config,
	db kvdb.Backend,
	blitzMetrics *metrics.FpMetrics,
	logger *zap.Logger,
) (*FinalityProviderApp, error) {

	em, err := eotsmanager.NewEOTSManagerClient(logger, cfg.EOTSManagerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "NewEOTSManagerClient failed")
	}

	cc, err := fpcc.NewBabylonController(fpConfig, logger)
	if err != nil {
		return nil, errors.Wrap(err, "NewBabylonController failed")
	}

	consumerCon, err := controllers.NewOrbitConsumerController(
		ctx, cfg, fpConfig.OPStackL2Config, logger,
	)
	if err != nil {
		return nil, errors.Wrap(err, "NewOrbitConsumerController failed")
	}

	l2Client, err := l2eth.NewL2EthClient(ctx, &cfg.Layer2)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create l2 eth client")
	}

	// TODO: use a simple service
	rpc := rpc.NewJsonRpcServer(logger, l2Client, cfg.Common.RpcVhosts, cfg.Common.RpcCors)

	return NewFinalityProviderApp(
		fpConfig, cc, consumerCon, em,
		db, blitzMetrics,
		rpc,
		cfg.Common.RpcServerIpPortAddress,
		logger,
	)
}

func NewFinalityProviderApp(
	config *fpcfg.Config,
	cc ccapi.ClientController, // TODO: this should be renamed as client controller is always going to be babylon
	consumerCon ccapi.ConsumerController,
	em fpeotsmanager.EOTSManager,
	db kvdb.Backend,
	blitzMetrics *metrics.FpMetrics,
	rpc *rpc.JsonRpcServer,
	jsonRpcServerIpPortAddr string,
	logger *zap.Logger,
) (*FinalityProviderApp, error) {
	fpStore, err := store.NewFinalityProviderStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate finality provider store: %w", err)
	}
	pubRandStore, err := store.NewPubRandProofStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate public randomness store: %w", err)
	}

	fpMetrics := fp_metrics.NewFpMetrics()
	fpm, err := NewFinalityProviderManager(fpStore, pubRandStore, config, cc, consumerCon, em, fpMetrics, blitzMetrics, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create finality-provider manager: %w", err)
	}

	return &FinalityProviderApp{
		fpManager:               fpm,
		config:                  config,
		eotsManager:             em,
		logger:                  logger,
		rpc:                     rpc,
		jsonRpcServerIpPortAddr: jsonRpcServerIpPortAddr,
		quit:                    make(chan struct{}),
	}, nil
}

func (app *FinalityProviderApp) GetAllStoredFinalityProviders() ([]*proto.FinalityProviderInfo, error) {
	return app.fpManager.AllFinalityProviders()
}

// Start starts only the finality-provider daemon without any finality-provider instances
func (app *FinalityProviderApp) Start(ctx context.Context, fpPk *bbntypes.BIP340PubKey, passphrase string) error {
	err := app.startImpl()
	if err != nil {
		return errors.Wrap(err, "start failed")
	}

	app.wg.Add(1)
	go func() {
		defer func() {
			app.logger.Debug("json RPC server stopped")
			app.wg.Done()
		}()

		app.rpc.StartServer(ctx, app.jsonRpcServerIpPortAddr)
	}()

	err = app.fpManager.StartFinalityProvider(fpPk, passphrase)
	if err != nil {
		return errors.Wrap(err, "StartHandlingFinalityProvider failed")
	}

	for {
		select {
		case <-ctx.Done():
			app.logger.Sugar().Info("app stop")
			err := app.stopImpl()
			if err != nil {
				app.logger.Sugar().Errorf("app stop failed: %v", err)
				return errors.Wrap(err, "stop failed")
			}
			return nil
		}
	}
}

func (app *FinalityProviderApp) Wait() {
	app.wg.Wait()
}

// Start starts only the finality-provider daemon without any finality-provider instances
func (app *FinalityProviderApp) startImpl() error {
	var startErr error
	app.startOnce.Do(func() {
		app.logger.Info("Starting FinalityProviderApp")
	})

	return startErr
}

func (app *FinalityProviderApp) stopImpl() error {
	var stopErr error
	app.stopOnce.Do(func() {
		app.logger.Info("Stopping FinalityProviderApp")

		// Always stop the submission loop first to not generate additional events and actions
		app.logger.Debug("Stopping submission loop")
		close(app.quit)
		app.wg.Wait()

		app.logger.Debug("Stopping finality providers")
		if app.fpManager.isStarted.Swap(true) {
			if err := app.fpManager.Stop(); err != nil {
				stopErr = err
				return
			}
		}

		app.logger.Debug("Stopping EOTS manager")
		if err := app.eotsManager.Close(); err != nil {
			stopErr = err
			return
		}

		app.logger.Debug("FinalityProviderApp successfully stopped")

	})
	return stopErr
}
