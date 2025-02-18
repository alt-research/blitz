package finalityprovider

import (
	"context"
	"fmt"
	"sync"

	"github.com/lightningnetwork/lnd/kvdb"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	bbnclient "github.com/babylonlabs-io/babylon/client/client"
	bbntypes "github.com/babylonlabs-io/babylon/types"
	fpcc "github.com/babylonlabs-io/finality-provider/clientcontroller"
	ccapi "github.com/babylonlabs-io/finality-provider/clientcontroller/api"
	fpeotsmanager "github.com/babylonlabs-io/finality-provider/eotsmanager"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/babylonlabs-io/finality-provider/finality-provider/proto"
	"github.com/babylonlabs-io/finality-provider/finality-provider/store"
	fp_metrics "github.com/babylonlabs-io/finality-provider/metrics"

	"github.com/alt-research/blitz/finality-gadget/client/eotsmanager"
	"github.com/alt-research/blitz/finality-gadget/metrics"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	"github.com/alt-research/blitz/finality-gadget/operator/finalityprovider/controllers"
	"github.com/alt-research/blitz/finality-gadget/sdk/cwclient"
)

type FinalityProviderApp struct {
	startOnce sync.Once
	stopOnce  sync.Once

	quit chan struct{}

	config *fpcfg.Config

	fpManager   *FinalityProviderManager
	eotsManager fpeotsmanager.EOTSManager
	logger      *zap.Logger

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

	return NewFinalityProviderApp(
		ctx, fpConfig, cc, consumerCon, em, db, blitzMetrics, logger,
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

	bbnConfig := fpcfg.BBNConfigToBabylonConfig(config.BabylonConfig)
	babylonClient, err := bbnclient.New(
		&bbnConfig,
		logger,
	)

	// Create cosmwasm client
	cwClient := cwclient.NewCosmWasmClient(
		babylonClient.QueryClient.RPCClient,
		config.OPStackL2Config.OPFinalityGadgetAddress)

	fpMetrics := fp_metrics.NewFpMetrics()
	fpm, err := NewFinalityProviderManager(
		ctx,
		fpStore, pubRandStore, config, cc,
		consumerCon, em, fpMetrics, blitzMetrics, cwClient, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create finality-provider manager: %w", err)
	}

	return &FinalityProviderApp{
		fpManager:   fpm,
		config:      config,
		eotsManager: em,
		logger:      logger,
		quit:        make(chan struct{}),
	}, nil
}

func (app *FinalityProviderApp) GetAllStoredFinalityProviders() ([]*proto.FinalityProviderInfo, error) {
	return app.fpManager.AllFinalityProviders()
}

// Start starts only the finality-provider daemon without any finality-provider instances
func (app *FinalityProviderApp) Start(ctx context.Context, fpPk *bbntypes.BIP340PubKey) error {
	err := app.startImpl()
	if err != nil {
		return errors.Wrap(err, "start failed")
	}

	err = app.fpManager.StartFinalityProvider(fpPk)
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
