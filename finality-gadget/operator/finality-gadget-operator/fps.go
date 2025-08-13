package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/urfave/cli"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	"github.com/alt-research/blitz/finality-gadget/operator/fp"
)

func fpsRestore(cliCtx *cli.Context) error {
	var config configs.OperatorConfig
	if err := utils.ReadConfig(cliCtx, defaultConfigPath, &config); err != nil {
		log.Fatalf("read config failed by %v", err)
		return err
	}
	config.WithEnv()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	zapLogger, err := logging.NewZapLoggerInner(logging.NewLogLevel(config.Common.Production))
	if err != nil {
		log.Fatalf("new logger failed by %v", err)
		return err
	}

	keyName := cliCtx.Args().Get(0)
	fpBtcPk := cliCtx.Args().Get(1)

	chainId := config.Layer2.ChainId

	zapLogger.Sugar().Infof("fp btc pk %v in %v", fpBtcPk, chainId)

	fpConfig, dbBackend, err := newAppParams(ctx, &config)
	if err != nil {
		return fmt.Errorf("failed to create params for app: %w", err)
	}

	app, err := fp.NewFpsCmdProvider(fpConfig.Common, dbBackend, zapLogger)
	if err != nil {
		return fmt.Errorf("failed to create cmd provider: %w", err)
	}

	return app.RestoreFP(ctx, keyName, strconv.Itoa(int(chainId)), fpBtcPk)
}

func fpsShow(cliCtx *cli.Context) error {
	var config configs.OperatorConfig
	if err := utils.ReadConfig(cliCtx, defaultConfigPath, &config); err != nil {
		log.Fatalf("read config failed by %v", err)
		return err
	}
	config.WithEnv()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	zapLogger, err := logging.NewZapLoggerInner(logging.NewLogLevel(config.Common.Production))
	if err != nil {
		log.Fatalf("new logger failed by %v", err)
		return err
	}

	fpConfig, dbBackend, err := newAppParams(ctx, &config)
	if err != nil {
		return fmt.Errorf("failed to create params for app: %w", err)
	}

	app, err := fp.NewFpsCmdProvider(fpConfig.Common, dbBackend, zapLogger)
	if err != nil {
		return fmt.Errorf("failed to create cmd provider: %w", err)
	}

	storedFps, err := app.ListAllFinalityProvidersInfo()
	if err != nil {
		return fmt.Errorf("failed to GetAllStoredFinalityProviders: %w", err)
	}

	for _, sfp := range storedFps {
		fmt.Printf("finality provider %v\n", sfp)
	}

	return nil
}
