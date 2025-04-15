package configs

import (
	"github.com/alt-research/blitz/finality-gadget/core/utils"
)

type Config struct {
	BBNChainID        string `yaml:"bbnchainid" description:"BabylonChain chain ID"`
	BBNRPCAddress     string `yaml:"bbnrpcaddress" description:"BabylonChain chain RPC address"`
	DBFilePath        string `yaml:"dbfilepath" description:"path to the DB file"`
	BitcoinRPCHost    string `yaml:"bitcoinrpchost" description:"rpc host address of the bitcoin node"`
	BitcoinRPCUser    string `yaml:"bitcoinrpcuser" description:"rpc user of the bitcoin node"`
	BitcoinRPCPass    string `yaml:"bitcoinrpcpass" description:"rpc password of the bitcoin node"`
	BitcoinDisableTLS bool   `yaml:"bitcoindisabletls" description:"disable TLS for RPC connections"`
	FGContractAddress string `yaml:"fgcontractaddress" description:"BabylonChain op finality gadget contract address"`
}

type BabylonConfig struct {
	FinalityGadgetCfg Config `yaml:"finality_gadget"`
}

func (c *BabylonConfig) WithEnv() {
	c.FinalityGadgetCfg.DBFilePath = utils.LookupEnvStr("FINALITY_GADGET_DB_FILE_PATH", c.FinalityGadgetCfg.DBFilePath)
	c.FinalityGadgetCfg.BBNChainID = utils.LookupEnvStr("FINALITY_GADGET_BABYLON_CHAINID", c.FinalityGadgetCfg.BBNChainID)
	c.FinalityGadgetCfg.BBNRPCAddress = utils.LookupEnvStr("FINALITY_GADGET_BABYLON_RPC_ADDR", c.FinalityGadgetCfg.BBNRPCAddress)
}
