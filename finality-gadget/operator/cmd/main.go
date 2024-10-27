package main

import (
	"log"
	"os"

	"github.com/carlmjohnson/versioninfo"
	"github.com/urfave/cli"

	"github.com/alt-research/blitz/finality-gadget/operator/configs"
)

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
			Name:    "keys",
			Aliases: []string{"k"},
			Usage:   "subcommand for keys",
			Subcommands: []cli.Command{
				{
					Name:   "restore",
					Usage:  "restore keys from mnemonic",
					Action: keysRestore,
				},
				{
					Name:  "show",
					Usage: "show address for key",
				},
			},
		},
		{
			Name:    "fps",
			Aliases: []string{"k"},
			Usage:   "subcommand for keys",
			Subcommands: []cli.Command{
				{
					Name:   "restore",
					Usage:  "restore keys from mnemonic",
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
