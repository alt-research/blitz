package configs

import (
	"cosmossdk.io/errors"
	"github.com/alt-research/blitz/finality-gadget/client/eotsmanager"
	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/core/configs"
	commonConfig "github.com/alt-research/blitz/finality-gadget/core/configs"
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type OperatorConfig struct {
	Common            commonConfig.CommonConfig `yaml:"common,omitempty"`
	Layer2            l2eth.Config              `yaml:"layer2,omitempty"`
	Babylon           configs.BabylonConfig     `yaml:"babylon,omitempty"`
	EOTSManagerConfig eotsmanager.Config        `yaml:"eotsManager,omitempty"`

	// fp home root path create by fpd.
	FinalityProviderHomePath string `yaml:"finalityProviderHomePath,omitempty"`
	// fp_addr is the bech32 chain address identifier of the finality provider.
	FpAddr string `yaml:"fp_addr,omitempty"`
	// btc_pk is the BTC secp256k1 PK of the finality provider encoded in BIP-340 spec
	BtcPk string `yaml:"btc_pk,omitempty"`
}

// use the env config first for some keys
func (c *OperatorConfig) WithEnv() {
	c.Common.WithEnv()
	c.Layer2.WithEnv()
	c.Babylon.WithEnv()
	c.EOTSManagerConfig.WithEnv()

	c.FinalityProviderHomePath = utils.LookupEnvStr("FINALITY_PROVIDER_HOME_PATH", c.FinalityProviderHomePath)
	c.FpAddr = utils.LookupEnvStr("FINALITY_PROVIDER_ADDRESS", c.FpAddr)
	c.BtcPk = utils.LookupEnvStr("FINALITY_PROVIDER_BTC_PK", c.BtcPk)

}

func (c *OperatorConfig) GetBtcPk() (*btcec.PublicKey, error) {
	btcPkBytes, err := hexutil.Decode(c.BtcPk)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid BTC public key hex string: %v", c.BtcPk)
	}

	btcPk, err := schnorr.ParsePubKey(btcPkBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid BTC public key: %w", c.BtcPk)
	}

	return btcPk, nil
}
