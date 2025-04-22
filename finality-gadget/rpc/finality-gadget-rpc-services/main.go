package main

import (
	"log"
	"os"

	"github.com/carlmjohnson/versioninfo"
	"github.com/urfave/cli"

	"github.com/alt-research/blitz/finality-gadget/operator/configs"
)

const defaultConfigPath = "./finality-gadget-rpc-services.yaml"

func main() {
	app := cli.NewApp()
	app.Flags = configs.Flags
	app.Version = versioninfo.Short()
	app.Name = "finality-gadget-rpc-services"
	app.Usage = "The finality-gadget rpc services"

	app.Action = rpcService
	app.Commands = []cli.Command{}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln("Application failed.", "Message:", err)
	}

}
