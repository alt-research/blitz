package configs

import (
	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/core/configs"
	commonConfig "github.com/alt-research/blitz/finality-gadget/core/configs"
	"github.com/alt-research/blitz/finality-gadget/operator/finalityprovider"
)

type OperatorConfig struct {
	Common           commonConfig.CommonConfig `yaml:"common,omitempty"`
	Layer2           l2eth.Config              `yaml:"layer2,omitempty"`
	Babylon          configs.BabylonConfig     `yaml:"babylon,omitempty"`
	FinalityProvider finalityprovider.Config   `yaml:"finalityProvider,omitempty"`
}

// use the env config first for some keys
func (c *OperatorConfig) WithEnv() {
	c.Common.WithEnv()
	c.Layer2.WithEnv()
	c.Babylon.WithEnv()
	c.FinalityProvider.WithEnv()
}
