package cwclient

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/btcsuite/btcd/btcec/v2"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
)

var _ ICosmosWasmContractClient = &CosmWasmClient{}

type CosmWasmClient struct {
	rpcclient.Client

	logger logging.Logger

	btcPk        *btcec.PublicKey
	btcPkHex     string
	contractAddr string
}

const (
	// hardcode the timeout to 20 seconds. We can expose it to the params once needed
	DefaultTimeout = 20 * time.Second
)

func NewCosmWasmClient(
	logger logging.Logger,
	rpcClient rpcclient.Client,
	btcPk *btcec.PublicKey,
	btcPkHex string,
	contractAddr string) *CosmWasmClient {
	return &CosmWasmClient{
		Client:       rpcClient,
		logger:       logger,
		btcPk:        btcPk,
		btcPkHex:     btcPkHex,
		contractAddr: contractAddr,
	}
}

// querySmartContractState queries the smart contract state given the contract address and query data
func (cwClient *CosmWasmClient) querySmartContractState(
	ctx context.Context,
	queryData []byte,
	resp any,
) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	sdkClientCtx := cosmosclient.Context{Client: cwClient.Client}
	wasmQueryClient := wasmtypes.NewQueryClient(sdkClientCtx)

	req := &wasmtypes.QuerySmartContractStateRequest{
		Address:   cwClient.contractAddr,
		QueryData: queryData,
	}
	respData, err := wasmQueryClient.SmartContractState(ctx, req)
	if err != nil {
		return errors.Wrap(err, "query smart contract state failed")
	}

	if err := json.Unmarshal(respData.Data, resp); err != nil {
		return errors.Wrap(err, "unmarshal smart contract state failed")
	}

	return nil
}
