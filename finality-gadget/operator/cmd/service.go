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

	cfg, err := fpcfg.LoadConfig(config.FinalityProviderHomePath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	cfg.NumPubRand = 1

	dbBackend, err := cfg.DatabaseConfig.GetDbBackend()
	if err != nil {
		return fmt.Errorf("failed to create db backend: %w", err)
	}

	logger, err := logging.NewZapLogger(logging.NewLogLevel(config.Common.Production))
	if err != nil {
		log.Fatalf("new logger failed by %v", err)
		return err
	}

	pk, err := config.GetBtcPk()
	if err != nil {
		log.Fatalf("GetBtcPk failed by %v", err)
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	zaplogger, err := logging.NewZapLoggerInner(logging.NewLogLevel(config.Common.Production))
	if err != nil {
		log.Fatalf("new logger failed by %v", err)
		return err
	}

	logger.Info("NewFinalityProviderAppFromConfig")

	app, err := finalityprovider.NewFinalityProviderAppFromConfig(ctx, &config, cfg, dbBackend, zaplogger)
	if err != nil {
		return errors.Wrap(err, "new provider failed")
	}

	err = app.Start(ctx, bbn.NewBIP340PubKeyFromBTCPK(pk), "")
	if err != nil {
		return errors.Wrap(err, "StartFinalityProviderInstance failed")
	}

	return nil
}
