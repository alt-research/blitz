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
	"github.com/alt-research/blitz/finality-gadget/operator"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
)

func main() {
	app := cli.NewApp()
	app.Flags = configs.Flags
	app.Version = versioninfo.Short()
	app.Name = "finality-gadget-operator"
	app.Usage = "The finality-gadget operator"
	app.Description = "Service that send the finality-gadget by FPs to babylon's contract"

	app.Action = operatorMain
	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln("Application failed.", "Message:", err)
	}
}

func operatorMain(cliCtx *cli.Context) error {
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

	logger.Info(
		"Finality gadget operator Start",
		"version", versioninfo.Version,
		"revision", versioninfo.Revision,
		"dirtyBuild", versioninfo.DirtyBuild,
		"lastCommit", versioninfo.LastCommit,
	)

	logger.Debug("configs", "cfg", config)

	operatorService, err := operator.NewFinalityGadgetOperatorService(ctx, &config, logger)
	if err != nil {
		log.Fatalln("Finality gadget operator new failed", "err", err.Error())
		return err
	}

	err = operatorService.Start(ctx)
	if err != nil {
		return errors.Wrap(err, "start Finality gadget operator failed")
	}

	operatorService.Wait()

	return nil
}
