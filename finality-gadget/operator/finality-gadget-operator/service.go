package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/carlmjohnson/versioninfo"
	"github.com/pkg/errors"
	"github.com/urfave/cli"

	rollupfpcfg "github.com/babylonlabs-io/finality-provider/bsn/rollup/config"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/alt-research/blitz/finality-gadget/metrics"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	"github.com/alt-research/blitz/finality-gadget/operator/fp"
)

func finalityProvider(cliCtx *cli.Context) error {
	var config configs.OperatorConfig
	if err := utils.ReadConfig(cliCtx, defaultConfigPath, &config); err != nil {
		log.Fatalf("read config failed by %v", err)
		return err
	}
	config.WithEnv()

	logger, err := logging.NewZapLogger(logging.NewLogLevel(config.Common.Production))
	if err != nil {
		log.Fatalf("new logger failed by %v", err)
		return err
	}

	logger.Infof("fp operator version %v", versioninfo.Short())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("NewFinalityProviderAppFromConfig")

	zaplogger, err := logging.NewZapLoggerInner(logging.NewLogLevel(config.Common.Production))
	if err != nil {
		log.Fatalf("new logger failed by %v", err)
		return err
	}
	promAddr, err := config.MetricsConfig.Address()
	if err != nil {
		return fmt.Errorf("failed to get prometheus address: %w", err)
	}
	metricsServer := metrics.Start(promAddr, zaplogger)
	defer metricsServer.Stop(context.Background())

	app, err := newApp(ctx, &config, metrics.NewFpMetrics())
	if err != nil {
		return errors.Wrap(err, "new provider failed")
	}

	err = app.Start(ctx, config.BtcPk)
	if err != nil {
		return errors.Wrap(err, "StartFinalityProviderInstance failed")
	}

	return nil
}

func newApp(ctx context.Context, config *configs.OperatorConfig, blitzMetrics *metrics.FpMetrics) (*fp.FinalityProviderApp, error) {
	fpConfig, err := rollupfpcfg.LoadConfig(config.FinalityProviderHomePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	zaplogger, err := logging.NewZapLoggerInner(logging.NewLogLevel(config.Common.Production))
	if err != nil {
		log.Fatalf("new logger failed by %v", err)
		return nil, err
	}

	dbBackend, err := fpConfig.Common.DatabaseConfig.GetDBBackend()
	if err != nil {
		return nil, fmt.Errorf("failed to create db backend: %w", err)
	}

	app, err := fp.NewFinalityProviderAppFromConfig(ctx, config, fpConfig, dbBackend, blitzMetrics, zaplogger)
	if err != nil {
		return nil, errors.Wrap(err, "new provider failed")
	}

	return app, nil
}
