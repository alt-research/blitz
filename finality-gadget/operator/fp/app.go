package fp

import (
	"context"
	"fmt"
	"sync"

	"github.com/lightningnetwork/lnd/kvdb"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/babylonlabs-io/babylon/types"
	fpcc "github.com/babylonlabs-io/finality-provider/clientcontroller"
	ccapi "github.com/babylonlabs-io/finality-provider/clientcontroller/api"
	fpeotsmanager "github.com/babylonlabs-io/finality-provider/eotsmanager"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/babylonlabs-io/finality-provider/finality-provider/proto"
	"github.com/babylonlabs-io/finality-provider/finality-provider/service"
	fp_metrics "github.com/babylonlabs-io/finality-provider/metrics"

	"github.com/alt-research/blitz/finality-gadget/client/eotsmanager"
	"github.com/alt-research/blitz/finality-gadget/metrics"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	"github.com/alt-research/blitz/finality-gadget/operator/fp/controllers"
	"github.com/alt-research/blitz/finality-gadget/rpc"
)

type FinalityProviderApp struct {
	stopOnce sync.Once

	quit chan struct{}

	config *fpcfg.Config

	fpApp       *service.FinalityProviderApp
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
	logger *zap.Logger,
) (*FinalityProviderApp, error) {
	blitzMetrics := metrics.NewFpMetrics()

	em, err := eotsmanager.NewEOTSManagerClient(logger, cfg.EOTSManagerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "NewEOTSManagerClient failed")
	}

	cc, err := fpcc.NewBabylonController(fpConfig, logger)
	if err != nil {
		return nil, errors.Wrap(err, "NewBabylonController failed")
	}

	consumerCon, err := controllers.NewOrbitConsumerController(
		ctx, cfg, fpConfig.OPStackL2Config, blitzMetrics, logger,
	)
	if err != nil {
		return nil, errors.Wrap(err, "NewOrbitConsumerController failed")
	}

	var rpcServer *rpc.JsonRpcServer

	if cfg.Common.RpcServerIpPortAddress != "" {
		rpcServer, err = rpc.NewJsonRpcServer(ctx, logger, cfg, fpConfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create NewJsonRpcServer")
		}
	}

	return NewFinalityProviderApp(
		ctx,
		fpConfig, cc, consumerCon, em,
		db, blitzMetrics,
		rpcServer,
		cfg.Common.RpcServerIpPortAddress,
		logger,
	)
}

func NewFinalityProviderApp(
	ctx context.Context,
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
	fpMetrics := fp_metrics.NewFpMetrics()
	poller := service.NewChainPoller(logger, config.PollerConfig, consumerCon, fpMetrics)
	fpApp, err := service.NewFinalityProviderApp(
		config,
		cc,
		consumerCon,
		em,
		poller,
		fpMetrics, db, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create finality-provider manager: %w", err)
	}

	return &FinalityProviderApp{
		fpApp:                   fpApp,
		config:                  config,
		eotsManager:             em,
		logger:                  logger,
		rpc:                     rpc,
		jsonRpcServerIpPortAddr: jsonRpcServerIpPortAddr,
		quit:                    make(chan struct{}),
	}, nil
}

func (app *FinalityProviderApp) GetAllStoredFinalityProviders() ([]*proto.FinalityProviderInfo, error) {
	return app.fpApp.ListAllFinalityProvidersInfo()
}

// Start starts only the finality-provider daemon without any finality-provider instances
func (app *FinalityProviderApp) Start(ctx context.Context, fpPkStr string) error {
	// only start the app without starting any finality provider instance
	// this is needed for new finality provider registration or unjailing
	// finality providers
	if err := app.fpApp.Start(); err != nil {
		return fmt.Errorf("failed to start the finality provider app: %w", err)
	}

	if app.jsonRpcServerIpPortAddr != "" && app.rpc != nil {
		app.wg.Add(1)
		go func() {
			defer func() {
				app.logger.Debug("json RPC server stopped")
				app.wg.Done()
			}()

			app.rpc.StartServer(ctx, app.jsonRpcServerIpPortAddr)
		}()
	}

	// fp instance will be started if public key is specified
	if fpPkStr != "" {
		// start the finality-provider instance with the given public key
		fpPk, err := types.NewBIP340PubKeyFromHex(fpPkStr)
		if err != nil {
			return fmt.Errorf("invalid finality provider public key %s: %w", fpPkStr, err)
		}

		if err := app.fpApp.StartFinalityProvider(fpPk); err != nil {
			return fmt.Errorf("failed to start by fpPkStr %s: %w", fpPkStr, err)
		}
	} else {
		app.logger.Sugar().Info("start fp by storedFps")
		storedFps, err := app.fpApp.GetFinalityProviderStore().GetAllStoredFinalityProviders()
		if err != nil {
			return err
		}

		if len(storedFps) == 1 {
			if err := app.fpApp.StartFinalityProvider(types.NewBIP340PubKeyFromBTCPK(storedFps[0].BtcPk)); err != nil {
				return fmt.Errorf("failed to start by storedFps %s: %w", storedFps[0].BtcPk, err)
			}
		}

		if len(storedFps) > 1 {
			return fmt.Errorf("%d finality providers found in DB. Please specify the EOTS public key", len(storedFps))
		}
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

func (app *FinalityProviderApp) stopImpl() error {
	var stopErr error
	app.stopOnce.Do(func() {
		app.logger.Info("Stopping FinalityProviderApp")

		// Always stop the submission loop first to not generate additional events and actions
		app.logger.Debug("Stopping submission loop")
		close(app.quit)
		app.wg.Wait()

		app.logger.Debug("Stopping finality providers")

		if err := app.fpApp.Stop(); err != nil {
			stopErr = err
			return
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
