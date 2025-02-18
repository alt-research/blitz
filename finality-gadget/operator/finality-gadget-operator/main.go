package main

import (
	"log"
	"os"

	"github.com/carlmjohnson/versioninfo"
	"github.com/urfave/cli"

	"github.com/alt-research/blitz/finality-gadget/operator/configs"
)

const defaultConfigPath = "./finality-gadget-operator.yaml"

func main() {
	app := cli.NewApp()
	app.Flags = configs.Flags
	app.Version = versioninfo.Short()
	app.Name = "finality-gadget-operator"
	app.Usage = "The finality-gadget operator"
	app.Description = "Service that send the finality-gadget by FPs to babylon's contract"

	app.Action = finalityProvider
	app.Commands = []cli.Command{
		{
			Name:    "fps",
			Aliases: []string{"k"},
			Usage:   "subcommand for finality provider manage",
			Subcommands: []cli.Command{
				{
					Name:   "restore",
					Usage:  "restore a fp as submitter",
					Action: fpsRestore,
				},
				{
					Name:   "show",
					Usage:  "show address for key",
					Action: fpsShow,
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln("Application failed.", "Message:", err)
	}

}
