package main

import (
	"log"
	"os"

	"github.com/carlmjohnson/versioninfo"
	"github.com/urfave/cli"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/alt-research/blitz/finality-gadget/signer/configs"
)

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
		"Finality gadget signer Start",
		"version", versioninfo.Version,
		"revision", versioninfo.Revision,
		"dirtyBuild", versioninfo.DirtyBuild,
		"lastCommit", versioninfo.LastCommit,
	)

	return nil
}
