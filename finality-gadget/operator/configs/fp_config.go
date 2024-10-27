package configs

import (
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
)

func ToCosmosProviderConfig(cfg *fpcfg.OPStackL2Config) cosmos.CosmosProviderConfig {
	return cosmos.CosmosProviderConfig{
		Key:            cfg.Key,
		ChainID:        cfg.ChainID,
		RPCAddr:        cfg.RPCAddr,
		AccountPrefix:  cfg.AccountPrefix,
		KeyringBackend: cfg.KeyringBackend,
		GasAdjustment:  cfg.GasAdjustment,
		GasPrices:      cfg.GasPrices,
		KeyDirectory:   cfg.KeyDirectory,
		Debug:          cfg.Debug,
		Timeout:        cfg.Timeout.String(),
		BlockTimeout:   cfg.BlockTimeout.String(),
		OutputFormat:   cfg.OutputFormat,
		SignModeStr:    cfg.SignModeStr,
	}
}
