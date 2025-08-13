package configs

import (
	"github.com/alt-research/blitz/finality-gadget/client/eotsmanager"
	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/core/configs"
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/alt-research/blitz/finality-gadget/metrics"
)

type OperatorConfig struct {
	Common            configs.CommonConfig  `yaml:"common,omitempty"`
	Layer2            l2eth.Config          `yaml:"layer2,omitempty"`
	Babylon           configs.BabylonConfig `yaml:"babylon,omitempty"`
	EOTSManagerConfig eotsmanager.Config    `yaml:"eotsManager,omitempty"`
	MetricsConfig     metrics.Config        `yaml:"metrics,omitempty"`

	// fp home root path create by fpd.
	FinalityProviderHomePath string `yaml:"finalityProviderHomePath,omitempty"`
	// btc_pk is the BTC secp256k1 PK of the finality provider encoded in BIP-340 spec
	BtcPk string `yaml:"btc_pk,omitempty"`
}

// use the env config first for some keys
func (c *OperatorConfig) WithEnv() {
	c.Common.WithEnv()
	c.Layer2.WithEnv()
	c.Babylon.WithEnv()
	c.EOTSManagerConfig.WithEnv()
	c.MetricsConfig.WithEnv()

	c.FinalityProviderHomePath = utils.LookupEnvStr("FINALITY_PROVIDER_HOME_PATH", c.FinalityProviderHomePath)
	c.BtcPk = utils.LookupEnvStr("FINALITY_PROVIDER_BTC_PK", c.BtcPk)

}
