package configs

import (
	"github.com/urfave/cli"

	"github.com/alt-research/blitz/finality-gadget/core/utils"
)

func init() {
	Flags = utils.Flags
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag
