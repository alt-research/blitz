package finalityprovider

import (
	"fmt"
	"net/url"
	"time"

	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
)

type Config struct {
	// The number of Schnorr public randomness for each commitment
	NumPubRand uint64 `yaml:"num_pub_rand,omitempty"`
	// The upper bound of the number of Schnorr public randomness for each commitment
	NumPubRandMax uint64 `yaml:"num_pub_rand_max,omitempty"`
	// The minimum gap between the last committed rand height and the current Babylon block height
	MinRandHeightGap uint32 `yaml:"minrandheightgap,omitempty"`
	// fp_addr is the bech32 chain address identifier of the finality provider.
	FpAddr string `yaml:"fp_addr,omitempty"`
	// btc_pk is the BTC secp256k1 PK of the finality provider encoded in BIP-340 spec
	BtcPk string `yaml:"btc_pk,omitempty"`
	// BabylonChain op finality gadget contract address
	FgContractAddress string `yaml:"fg_contract_address,omitempty"`
	// BabylonChain chain ID
	BbnChainID string `yaml:"bbn_chain_id,omitempty"`
	// BabylonChain chain RPC address
	BbnRpcAddress string `yaml:"bbn_rpc_address,omitempty"`
	// The cosmwasm config
	Cosmwasm CosmwasmConfig `yaml:"cosmwasm,omitempty"`
	// the consumer id
	ConsumerId string `yaml:"consumer_id,omitempty"`
}

func (c *Config) WithEnv() {
	c.FpAddr = utils.LookupEnvStr("FINALITY_PROVIDER_ADDRESS", c.FpAddr)
	c.BtcPk = utils.LookupEnvStr("FINALITY_PROVIDER_BTC_PK", c.BtcPk)
	c.FgContractAddress = utils.LookupEnvStr("FINALITY_PROVIDER_FG_CONTRACT_ADDRESS", c.FgContractAddress)
	c.BbnChainID = utils.LookupEnvStr("BBN_CHAIN_ID", c.BbnChainID)
	c.BbnRpcAddress = utils.LookupEnvStr("BBN_RPC_ADDRESS", c.BbnRpcAddress)
}

func (c *Config) GetBtcPk() (*btcec.PublicKey, error) {
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

func (cfg *Config) ToCosmosProviderConfig() cosmos.CosmosProviderConfig {
	return cfg.Cosmwasm.ToCosmosProviderConfig()
}

// CosmwasmConfig defines configuration for the Babylon client
// adapted from https://github.com/strangelove-ventures/lens/blob/v0.5.1/client/config.go
type CosmwasmConfig struct {
	Key              string        `yaml:"key"`
	ChainID          string        `yaml:"chain_id,omitempty"`
	RPCAddr          string        `yaml:"rpc_addr,omitempty"`
	GRPCAddr         string        `yaml:"grpc_addr,omitempty"`
	AccountPrefix    string        `yaml:"account_prefix,omitempty"`
	KeyringBackend   string        `yaml:"keyring_backend,omitempty"`
	GasAdjustment    float64       `yaml:"gas_adjustment,omitempty"`
	GasPrices        string        `yaml:"gas_prices,omitempty"`
	KeyDirectory     string        `yaml:"key_directory,omitempty"`
	Debug            bool          `yaml:"debug,omitempty"`
	Timeout          time.Duration `yaml:"timeout,omitempty"`
	BlockTimeout     time.Duration `yaml:"block_timeout,omitempty"`
	OutputFormat     string        `yaml:"output_format,omitempty"`
	SignModeStr      string        `yaml:"sign_mode,omitempty"`
	SubmitterAddress string        `yaml:"submitter_address,omitempty"`
}

func (cfg *CosmwasmConfig) Validate() error {
	if _, err := url.Parse(cfg.RPCAddr); err != nil {
		return fmt.Errorf("rpc-addr is not correctly formatted: %w", err)
	}

	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if cfg.BlockTimeout < 0 {
		return fmt.Errorf("block-timeout can't be negative")
	}

	return nil
}

func (cfg *CosmwasmConfig) ToCosmosProviderConfig() cosmos.CosmosProviderConfig {
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
