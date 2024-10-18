package finalityprovider

import (
	"context"
	"os"

	"cosmossdk.io/log"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	simsutils "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/encoding"

	bbnapp "github.com/babylonlabs-io/babylon/app"
	appparams "github.com/babylonlabs-io/babylon/app/params"
	bbn "github.com/babylonlabs-io/babylon/types"
)

// TmpAppOptions returns an app option with tmp dir and btc network
func TmpAppOptions() simsutils.AppOptionsMap {
	dir, err := os.MkdirTemp("", "babylon-tmp-app")
	if err != nil {
		panic(err)
	}
	appOpts := simsutils.AppOptionsMap{
		flags.FlagHome:       dir,
		"btc-config.network": string(bbn.BtcSimnet),
	}
	return appOpts
}

func NewTmpBabylonApp() *bbnapp.BabylonApp {
	signer, _ := bbnapp.SetupTestPrivSigner()
	app := bbnapp.NewBabylonApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		map[int64]bool{},
		0,
		signer,
		TmpAppOptions(),
		[]wasmkeeper.Option{})

	return app
}

// GetEncodingConfig returns a *registered* encoding config
// Note that the only way to register configuration is through the app creation
func GetEncodingConfig() *appparams.EncodingConfig {
	tmpApp := NewTmpBabylonApp()
	return &appparams.EncodingConfig{
		InterfaceRegistry: tmpApp.InterfaceRegistry(),
		Codec:             tmpApp.AppCodec(),
		TxConfig:          tmpApp.TxConfig(),
		Amino:             tmpApp.LegacyAmino(),
	}
}

// grpcProtoCodec is the implementation of the gRPC proto codec.
type grpcProtoCodec struct {
	cdc encoding.Codec
}

func (g grpcProtoCodec) Marshal(v interface{}) ([]byte, error) {
	return g.cdc.Marshal(v)
}

func (g grpcProtoCodec) Unmarshal(data []byte, v interface{}) error {
	return g.cdc.Unmarshal(data, v)
}

func (g grpcProtoCodec) Name() string {
	return "proto"
}

func init() {
	encCfg := GetEncodingConfig()

	cosmosCdc := cosmos.Codec{
		InterfaceRegistry: encCfg.InterfaceRegistry,
		Marshaler:         encCfg.Codec,
		TxConfig:          encCfg.TxConfig,
		Amino:             encCfg.Amino,
	}

	grpcCodec := codec.NewProtoCodec(cosmosCdc.InterfaceRegistry).GRPCCodec()
	grpcCodecWrapper := grpcProtoCodec{
		cdc: grpcCodec,
	}
	encoding.RegisterCodec(grpcCodecWrapper)
}

func NewProvider(ctx context.Context, cfg *Config, zaplogger *zap.Logger) (*cosmos.CosmosProvider, error) {
	cpCfg := cfg.ToCosmosProviderConfig()
	cpCfg.SigningAlgorithm = "secp256k1"
	// TODO: should add this to avoid https://github.com/cosmos/relayer/issues/1373
	cpCfg.ExtraCodecs = append(cpCfg.ExtraCodecs, "ethermint")

	provider, err := cpCfg.NewProvider(
		zaplogger,
		"", // TODO: set home path
		true,
		cfg.BbnChainID,
	)
	if err != nil {
		return nil, errors.Wrap(err, "new provider failed")
	}

	// Create tmp Babylon app to retrieve and register codecs
	// Need to override this manually as otherwise option from config is ignored
	encCfg := GetEncodingConfig()

	cp := provider.(*cosmos.CosmosProvider)
	cp.PCfg.KeyDirectory = cfg.Cosmwasm.KeyDirectory

	cosmosCdc := cosmos.Codec{
		InterfaceRegistry: encCfg.InterfaceRegistry,
		Marshaler:         encCfg.Codec,
		TxConfig:          encCfg.TxConfig,
		Amino:             encCfg.Amino,
	}
	cp.Cdc = cosmosCdc

	// initialise Cosmos provider
	// NOTE: this will create a RPC client. The RPC client will be used for
	// submitting txs and making ad hoc queries. It won't create WebSocket
	// connection with wasmd node
	err = cp.Init(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "init provider failed")
	}

	return cp, nil
}
