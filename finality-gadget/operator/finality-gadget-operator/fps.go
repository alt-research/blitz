package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
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

	zaplogger, err := logging.NewZapLoggerInner(logging.NewLogLevel(config.Common.Production))
	if err != nil {
		log.Fatalf("new logger failed by %v", err)
		return err
	}

	keyName := cliCtx.Args().Get(0)
	fpBtcPk := cliCtx.Args().Get(1)

	chainId := config.Layer2.ChainId

	zaplogger.Sugar().Infof("fp btc pk %v in %v", fpBtcPk, chainId)

	app, err := newApp(ctx, &config, nil)
	if err != nil {
		return errors.Wrap(err, "new provider failed")
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

	app, err := newApp(ctx, &config, nil)
	if err != nil {
		return errors.Wrap(err, "new provider failed")
	}

	storedFps, err := app.GetAllStoredFinalityProviders()
	if err != nil {
		return errors.Wrap(err, "GetAllStoredFinalityProviders failed")
	}

	for _, sfp := range storedFps {
		fmt.Printf("finality provider %v\n", sfp)
	}

	return nil
}
