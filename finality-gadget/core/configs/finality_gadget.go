package configs

import (
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	finalityGadgetConfig "github.com/babylonlabs-io/finality-gadget/config"
)

type BabylonConfig struct {
	FinalityGadgetCfg finalityGadgetConfig.Config `yaml:"finality_gadget"`
}

func (c *BabylonConfig) WithEnv() {
	c.FinalityGadgetCfg.DBFilePath = utils.LookupEnvStr("FINALITY_GADGET_DB_FILE_PATH", c.FinalityGadgetCfg.DBFilePath)
	c.FinalityGadgetCfg.BBNChainID = utils.LookupEnvStr("FINALITY_GADGET_BABYLON_CHAINID", c.FinalityGadgetCfg.BBNChainID)
	c.FinalityGadgetCfg.BBNRPCAddress = utils.LookupEnvStr("FINALITY_GADGET_BABYLON_RPC_ADDR", c.FinalityGadgetCfg.BBNRPCAddress)
}

func (c *BabylonConfig) FinalityGadget() *finalityGadgetConfig.Config {
	return &c.FinalityGadgetCfg
}
