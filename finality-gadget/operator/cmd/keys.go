package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"cosmossdk.io/errors"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/urfave/cli"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	"github.com/alt-research/blitz/finality-gadget/operator/finalityprovider/cosmosprovider"
)

func keysRestore(cliCtx *cli.Context) error {
	keyName := cliCtx.Args().Get(0)
	mnemonic := cliCtx.Args().Get(1)

	var config configs.OperatorConfig
	if err := utils.ReadConfig(cliCtx, &config); err != nil {
		log.Fatalf("read config failed by %v", err)
		return err
	}
	config.WithEnv()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fpConfig, err := fpcfg.LoadConfig(config.FinalityProviderHomePath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	zaplogger, err := logging.NewZapLoggerInner(logging.NewLogLevel(config.Common.Production))
	if err != nil {
		log.Fatalf("new logger failed by %v", err)
		return err
	}

	opCfg := fpConfig.OPStackL2Config

	provider, err := cosmosprovider.NewCosmosProvider(ctx, opCfg, zaplogger)
	if err != nil {
		return err
	}

	// zaplogger.Sugar().Infof("key exists %v", provider.KeyExists(keyName))

	if provider.KeyExists(keyName) {
		return fmt.Errorf("the key %v already exists", keyName)
	}

	// TODO: use flag
	coinType := uint32(118)

	address, err := provider.RestoreKey(keyName, mnemonic, coinType, provider.PCfg.SigningAlgorithm)
	if err != nil {
		return errors.Wrap(err, "failed to restore key")
	}

	fmt.Printf("restore key: %s\n", address)

	return nil
}
