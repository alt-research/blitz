package fp

import (
	"context"
	"sync"

	"github.com/lightningnetwork/lnd/kvdb"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	bbntypes "github.com/babylonlabs-io/babylon/types"
	ccapi "github.com/babylonlabs-io/finality-provider/clientcontroller/api"
	"github.com/babylonlabs-io/finality-provider/clientcontroller/babylon"
	fpeotsmanager "github.com/babylonlabs-io/finality-provider/eotsmanager"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/babylonlabs-io/finality-provider/finality-provider/service"

	"github.com/alt-research/blitz/finality-gadget/client/eotsmanager"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	"github.com/alt-research/blitz/finality-gadget/operator/fp/controllers"
)

type FinalityProviderApp struct {
	inner  *service.FinalityProviderApp
	logger *zap.Logger

	wg sync.WaitGroup
}

func NewFinalityProviderAppFromConfig(
	ctx context.Context,
	cfg *configs.OperatorConfig,
	fpConfig *fpcfg.Config,
	db kvdb.Backend,
	logger *zap.Logger,
) (*FinalityProviderApp, error) {

	em, err := eotsmanager.NewEOTSManagerClient(logger, cfg.EOTSManagerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "NewEOTSManagerClient failed")
	}

	cc, err := babylon.NewBabylonController(fpConfig.BabylonConfig, &fpConfig.BTCNetParams, logger)
	if err != nil {
		return nil, errors.Wrap(err, "NewBabylonController failed")
	}

	consumerCon, err := controllers.NewCosmwasmConsumerController(
		ctx, cfg, fpConfig, logger,
	)

	return NewFinalityProviderApp(
		fpConfig, cc, consumerCon, em, db, logger,
	)
}

func NewFinalityProviderApp(
	config *fpcfg.Config,
	cc ccapi.ClientController, // TODO: this should be renamed as client controller is always going to be babylon
	consumerCon ccapi.ConsumerController,
	em fpeotsmanager.EOTSManager,
	db kvdb.Backend,
	logger *zap.Logger,
) (*FinalityProviderApp, error) {
	app, err := service.NewFinalityProviderApp(config, cc, consumerCon, em, db, logger)
	if err != nil {
		return nil, errors.Wrap(err, "NewFinalityProviderApp failed")
	}

	return &FinalityProviderApp{
		inner:  app,
		logger: logger,
	}, nil
}

// Start starts only the finality-provider daemon without any finality-provider instances
func (app *FinalityProviderApp) Start(ctx context.Context, fpPk *bbntypes.BIP340PubKey, passphrase string) error {
	err := app.inner.Start()
	if err != nil {
		return errors.Wrap(err, "start failed")
	}

	err = app.inner.StartHandlingFinalityProvider(fpPk, passphrase)
	if err != nil {
		return errors.Wrap(err, "StartHandlingFinalityProvider failed")
	}

	for {
		select {
		case <-ctx.Done():
			app.logger.Sugar().Info("app stop")
			err := app.inner.Stop()
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
