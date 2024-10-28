package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/carlmjohnson/versioninfo"
	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/alt-research/blitz/finality-gadget/signer"
	"github.com/alt-research/blitz/finality-gadget/signer/configs"
)

const defaultConfigPath = "./finality-gadget-signer.yaml"

func main() {
	app := cli.NewApp()
	app.Flags = configs.Flags
	app.Version = versioninfo.Short()
	app.Name = "finality-gadget-signer"
	app.Usage = "The finality-gadget signer"
	app.Description = "Service that sign the finality-gadget commit by FPs to babylon's contract"

	app.Action = signerMain
	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln("Application failed.", "Message:", err)
	}
}

func signerMain(cliCtx *cli.Context) error {
	var config configs.SignerConfig
	if err := utils.ReadConfig(cliCtx, defaultConfigPath, &config); err != nil {
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

	logger.Info(
		"Finality gadget signer Start",
		"version", versioninfo.Version,
		"revision", versioninfo.Revision,
		"dirtyBuild", versioninfo.DirtyBuild,
		"lastCommit", versioninfo.LastCommit,
	)

	logger.Debug("configs", "cfg", config)

	signerService, err := signer.NewFinalityGadgetSignerService(ctx, &config, logger)
	if err != nil {
		log.Fatalln("Finality gadget signer new failed", "err", err.Error())
		return err
	}

	err = signerService.Start(ctx)
	if err != nil {
		return errors.Wrap(err, "start Finality gadget signer failed")
	}

	signerService.Wait()

	return nil
}
