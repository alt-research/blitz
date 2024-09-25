package main

import (
	"log"
	"os"

	"github.com/carlmjohnson/versioninfo"
	"github.com/urfave/cli"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/core/utils"
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

	logger, err := logging.NewZapLogger(logging.NewLogLevel(config.Production))
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

	return nil
}
