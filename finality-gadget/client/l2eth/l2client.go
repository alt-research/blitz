package l2eth

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

type L2EthClient struct {
	cfg *Config
	cli *ethclient.Client
}

type Config struct {
	// The eth rpc url
	EthRpcUrl string `yaml:"eth_rpc_url"`
	// The chain id of l2
	ChainId uint64 `yaml:"chain_id"`
}

func NewL2EthClient(ctx context.Context, cfg *Config) (*L2EthClient, error) {
	// Create L2 client
	cli, err := ethclient.Dial(cfg.EthRpcUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create L2 eth client by %s", cfg.EthRpcUrl)
	}

	// Check if chain id is expected
	chainId, err := cli.ChainID(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to got chain id from %s", cfg.EthRpcUrl)
	}

	if cfg.ChainId != 0 {
		if cfg.ChainId != chainId.Uint64() {
			return nil, errors.Errorf(
				"the chain id from %s expected %d, got %d",
				cfg.EthRpcUrl,
				cfg.ChainId, chainId.Uint64())
		}
	} else {
		cfg.ChainId = chainId.Uint64()
	}

	return &L2EthClient{
		cfg: cfg,
		cli: cli,
	}, nil
}
