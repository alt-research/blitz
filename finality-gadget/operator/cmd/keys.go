package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
)

func keysRestore(cliCtx *cli.Context) error {
	var config configs.OperatorConfig
	if err := utils.ReadConfig(cliCtx, &config); err != nil {
		log.Fatalf("read config failed by %v", err)
		return err
	}
	config.WithEnv()

	keyName := cliCtx.Args().Get(0)
	mnemonic := cliCtx.Args().Get(1)

	logger, err := logging.NewZapLogger(logging.NewLogLevel(config.Common.Production))
	if err != nil {
		log.Fatalf("new logger failed by %v", err)
		return err
	}

	logger.Debug("key restore", "name", keyName, "mnemonic", mnemonic)

	_, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// check the config file exists
	_, err = fpcfg.LoadConfig(config.FinalityProviderHomePath)
	if err != nil {
		return nil // config does not exist, so does not update it
	}

	return nil
}

func fpsShow(cliCtx *cli.Context) error {
	var config configs.OperatorConfig
	if err := utils.ReadConfig(cliCtx, &config); err != nil {
		log.Fatalf("read config failed by %v", err)
		return err
	}
	config.WithEnv()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := newApp(ctx, &config)
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
