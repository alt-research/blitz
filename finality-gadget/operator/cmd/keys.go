package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
	"github.com/alt-research/blitz/finality-gadget/operator/finalityprovider"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cp, err := finalityprovider.NewProvider(ctx, &config.FinalityProvider, logger.Inner())
	if err != nil {
		return errors.Wrap(err, "new provider failed")
	}

	if cp.KeyExists(keyName) {
		return errors.Errorf("the key %s already exists", keyName)
	}

	// TODO: use flag
	coinType := 118

	address, err := cp.RestoreKey(keyName, mnemonic, uint32(coinType), cp.PCfg.SigningAlgorithm)
	if err != nil {
		return err
	}

	logger.Info("restore key", "address", address)

	return nil
}

func keysShow(cliCtx *cli.Context) error {
	var config configs.OperatorConfig
	if err := utils.ReadConfig(cliCtx, &config); err != nil {
		log.Fatalf("read config failed by %v", err)
		return err
	}
	config.WithEnv()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger, err := logging.NewZapLogger(logging.NewLogLevel(config.Common.Production))
	if err != nil {
		log.Fatalf("new logger failed by %v", err)
		return err
	}

	keyName := cliCtx.Args().Get(0)

	logger.Info("show key", "name", keyName)

	cp, err := finalityprovider.NewProvider(ctx, &config.FinalityProvider, logger.Inner())
	if err != nil {
		return errors.Wrap(err, "new provider failed")
	}

	keyStore, err := cp.Keybase.Key(keyName)
	logger.Info("keystore", "store", keyStore, "err", err)

	if !cp.KeyExists(keyName) {
		return errors.Errorf("the key %s no exists", keyName)
	}

	key, err := cp.GetKeyAddressForKey(keyName)
	if err != nil {
		return errors.Wrapf(err, "failed to get key address for %v", keyName)
	}

	logger.Debug("key address", "name", keyName, "address", key)

	return nil
}
