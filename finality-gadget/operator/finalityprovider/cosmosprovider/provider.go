package cosmosprovider

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	wasmdparams "github.com/CosmWasm/wasmd/app/params"
	bbnapp "github.com/babylonlabs-io/babylon/app"
	"github.com/babylonlabs-io/babylon/app/params"
	"github.com/babylonlabs-io/babylon/client/babylonclient"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
)

func NewCosmosProvider(ctx context.Context, cfg *fpcfg.OPStackL2Config, zaplogger *zap.Logger) (*babylonclient.CosmosProvider, error) {
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
		"", // TODO: set home path
		cfg.ChainID,
	)

	if err != nil {
		return nil, err
	}
	cp, ok := provider.(*babylonclient.CosmosProvider)
	if !ok {
		return nil, fmt.Errorf("failed to cast provider to CosmosProvider")
	}
	cp.PCfg.KeyDirectory = cwConfig.KeyDirectory
	cp.Cdc = &params.EncodingConfig{
		InterfaceRegistry: cwEncodingCfg.InterfaceRegistry,
		Codec:             cwEncodingCfg.Codec,
		TxConfig:          cwEncodingCfg.TxConfig,
		Amino:             cwEncodingCfg.Amino,
	}

	// initialise Cosmos provider
	// NOTE: this will create a RPC client. The RPC client will be used for
	// submitting txs and making ad hoc queries. It won't create WebSocket
	// connection with wasmd node
	if err = cp.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize cosmos provider: %w", err)
	}

	return cp, nil
}
