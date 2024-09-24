package configs

import (
	commonConfig "github.com/alt-research/blitz/finality-gadget/core/configs"
)

type OperatorConfig struct {
	commonConfig.CommonConfig
}

// use the env config first for some keys
func (c *OperatorConfig) WithEnv() {
	c.CommonConfig.WithEnv()
}
