package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	bbn "github.com/babylonlabs-io/babylon/types"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	"github.com/alt-research/blitz/finality-gadget/operator/finalityprovider"
)

func finalityProvider(cliCtx *cli.Context) error {
	var config configs.OperatorConfig
	if err := utils.ReadConfig(cliCtx, &config); err != nil {
		log.Fatalf("read config failed by %v", err)
		return err
	}
	config.WithEnv()

	logger, err := logging.NewZapLogger(logging.NewLogLevel(config.Common.Production))
	if err != nil {
		log.Fatalf("new logger failed by %v", err)
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("NewFinalityProviderAppFromConfig")

	app, err := newApp(ctx, &config)
	if err != nil {
		return errors.Wrap(err, "new provider failed")
	}

	pk, err := config.GetBtcPk()
	if err != nil {
		log.Fatalf("GetBtcPk failed by %v", err)
		return err
	}

	err = app.Start(ctx, bbn.NewBIP340PubKeyFromBTCPK(pk), "")
	if err != nil {
		return errors.Wrap(err, "StartFinalityProviderInstance failed")
	}

	return nil
}

func newApp(ctx context.Context, config *configs.OperatorConfig) (*finalityprovider.FinalityProviderApp, error) {
	fpConfig, err := fpcfg.LoadConfig(config.FinalityProviderHomePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	fpConfig.NumPubRand = 1

	zaplogger, err := logging.NewZapLoggerInner(logging.NewLogLevel(config.Common.Production))
	if err != nil {
		log.Fatalf("new logger failed by %v", err)
		return nil, err
	}

	dbBackend, err := fpConfig.DatabaseConfig.GetDbBackend()
	if err != nil {
		return nil, fmt.Errorf("failed to create db backend: %w", err)
	}

	app, err := finalityprovider.NewFinalityProviderAppFromConfig(ctx, config, fpConfig, dbBackend, zaplogger)
	if err != nil {
		return nil, errors.Wrap(err, "new provider failed")
	}

	return app, nil
}
