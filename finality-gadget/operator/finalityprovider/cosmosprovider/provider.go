package cosmosprovider

import (
	"context"

	"go.uber.org/zap"

	wasmdparams "github.com/CosmWasm/wasmd/app/params"
	bbnapp "github.com/babylonlabs-io/babylon/app"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
)

func NewCosmosProvider(ctx context.Context, cfg *fpcfg.OPStackL2Config, zaplogger *zap.Logger) (*cosmos.CosmosProvider, error) {
	bbnEncodingCfg := bbnapp.GetEncodingConfig()
	cwEncodingCfg := wasmdparams.EncodingConfig{
		InterfaceRegistry: bbnEncodingCfg.InterfaceRegistry,
		Codec:             bbnEncodingCfg.Codec,
		TxConfig:          bbnEncodingCfg.TxConfig,
		Amino:             bbnEncodingCfg.Amino,
	}

	cwConfig := cfg.ToCosmwasmConfig()

	// ensure cfg is valid
	if err := cwConfig.Validate(); err != nil {
		return nil, err
	}

	provider, err := cwConfig.ToCosmosProviderConfig().NewProvider(
		zaplogger,
		"", // TODO: set home path
		true,
		cfg.ChainID,
	)

	if err != nil {
		return nil, err
	}
	cp := provider.(*cosmos.CosmosProvider)
	cp.PCfg.KeyDirectory = cwConfig.KeyDirectory
	cp.Cdc = cosmos.Codec{
		InterfaceRegistry: cwEncodingCfg.InterfaceRegistry,
		Marshaler:         cwEncodingCfg.Codec,
		TxConfig:          cwEncodingCfg.TxConfig,
		Amino:             cwEncodingCfg.Amino,
	}

	// initialise Cosmos provider
	// NOTE: this will create a RPC client. The RPC client will be used for
	// submitting txs and making ad hoc queries. It won't create WebSocket
	// connection with wasmd node
	err = cp.Init(ctx)
	if err != nil {
		return nil, err
	}

	return cp, nil
}
