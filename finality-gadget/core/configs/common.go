package configs

import "os"

type CommonConfig struct {
	// The service name
	Name string `yaml:"name"`
	// used to set the logger level (true = info, false = debug)
	Production             bool     `yaml:"production"`
	RpcServerIpPortAddress string   `yaml:"rpc_server_ip_port_address"`
	RpcVhosts              []string `yaml:"rpc_vhosts"`
	RpcCors                []string `yaml:"rpc_cors"`
}

// use the env config first for some keys
func (c *CommonConfig) WithEnv() {
	production, ok := os.LookupEnv("FINALITY_GADGET_PRODUCTION")
	if ok && production != "" {
		c.Production = production == "true"
	}

	name, ok := os.LookupEnv("FINALITY_GADGET_NAME")
	if ok && name != "" {
		c.Name = name
	}
}
