package configs

import (
	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	commonConfig "github.com/alt-research/blitz/finality-gadget/core/configs"
)

type OperatorConfig struct {
	Common commonConfig.CommonConfig `yaml:"common,omitempty"`
	Layer2 l2eth.Config              `yaml:"layer2,omitempty"`
}

// use the env config first for some keys
func (c *OperatorConfig) WithEnv() {
	c.Common.WithEnv()
	c.Layer2.WithEnv()
}
