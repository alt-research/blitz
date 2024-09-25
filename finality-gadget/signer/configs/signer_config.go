package configs

import (
	commonConfig "github.com/alt-research/blitz/finality-gadget/core/configs"
)

type SignerConfig struct {
	commonConfig.CommonConfig
}

// use the env config first for some keys
func (c *SignerConfig) WithEnv() {
	c.CommonConfig.WithEnv()
}
