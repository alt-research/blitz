package finalityprovider

import (
	"github.com/alt-research/blitz/finality-gadget/core/utils"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
)

type Config struct {
	// The number of Schnorr public randomness for each commitment
	NumPubRand uint64 `yaml:"num_pub_rand,omitempty"`
	// The upper bound of the number of Schnorr public randomness for each commitment
	NumPubRandMax uint64 `yaml:"num_pub_rand_max,omitempty"`
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
