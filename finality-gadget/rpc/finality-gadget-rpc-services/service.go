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

	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/alt-research/blitz/finality-gadget/metrics"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	"github.com/alt-research/blitz/finality-gadget/rpc"
)

func rpcService(cliCtx *cli.Context) error {
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

	rpc, err := newApp(ctx, &config, metrics.NewFpMetrics())
	if err != nil {
		return errors.Wrap(err, "new provider failed")
	}

	rpc.StartServer(ctx, config.Common.RpcServerIpPortAddress)

	return nil
}

func newApp(ctx context.Context, config *configs.OperatorConfig, blitzMetrics *metrics.FpMetrics) (*rpc.JsonRpcServer, error) {
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

	rpcApp, err := rpc.NewJsonRpcServer(ctx, zaplogger, config, fpConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create NewJsonRpcServer")
	}

	return rpcApp, nil
}
