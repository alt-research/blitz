package configs

import (
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/alt-research/blitz/finality-gadget/sdk/btcclient"
	sdkConfig "github.com/alt-research/blitz/finality-gadget/sdk/config"
)

type BabylonConfig struct {
	BTCConfig    btcclient.BTCConfig `yaml:"btc_config,omitempty"`
	ContractAddr string              `yaml:"contract_address,omitempty"`
	ChainID      string              `yaml:"chain_id,omitempty"`
	RPCAddr      string              `yaml:"rpc_address,omitempty"`
}

func (c *BabylonConfig) WithEnv() {
	c.ContractAddr = utils.LookupEnvStr("FINALITY_GADGET_BABYLON_CONTRACT_ADDR", c.ContractAddr)
	c.ChainID = utils.LookupEnvStr("FINALITY_GADGET_BABYLON_CHAINID", c.ChainID)
	c.RPCAddr = utils.LookupEnvStr("FINALITY_GADGET_BABYLON_RPC_ADDR", c.RPCAddr)
}

func (c *BabylonConfig) ToSdkConfig() *sdkConfig.Config {
	return &sdkConfig.Config{
		BTCConfig:    &c.BTCConfig,
		ContractAddr: c.ContractAddr,
		ChainID:      c.ChainID,
		RPCAddr:      c.RPCAddr,
	}
}
