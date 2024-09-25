package utils

import (
	"github.com/urfave/cli"
)

var (
	/* Required Flags */
	ConfigFileFlag = cli.StringFlag{
		Name:     "config",
		Required: false,
		Usage:    "Load configuration from `FILE`",
		EnvVar:   "FINALITY_GADGET_CONFIG_PATH",
	}
)

var requiredFlags = []cli.Flag{
	ConfigFileFlag,
}

var optionalFlags = []cli.Flag{}

func init() {
	Flags = append(requiredFlags, optionalFlags...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag
