package cwclient

import (
	"context"
	"encoding/json"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

var _ ICosmosWasmContractClient = &CosmWasmClient{}

type CosmWasmClient struct {
	rpcclient.Client
	contractAddr string
}

const (
	// hardcode the timeout to 20 seconds. We can expose it to the params once needed
	DefaultTimeout = 20 * time.Second
)

func NewCosmWasmClient(rpcClient rpcclient.Client, contractAddr string) *CosmWasmClient {
	return &CosmWasmClient{
		Client:       rpcClient,
		contractAddr: contractAddr,
	}
}

func (cwClient *CosmWasmClient) QueryListOfVotedFinalityProviders(
	ctx context.Context,
	height uint64,
	hash common.Hash,
) ([]string, error) {
	queryData, err := newQueryBlockVotersMsg(height, hash)
	if err != nil {
		return nil, err
	}

	votedFpPkHexList := []string{}
	if err := cwClient.querySmartContractState(
		ctx,
		queryData,
		&votedFpPkHexList); err != nil {
		return nil, err
	}

	return votedFpPkHexList, nil
}

func (cwClient *CosmWasmClient) QueryConfig(ctx context.Context) (contractConfigResponse, error) {
	queryData, err := newQueryConfigMsg()
	if err != nil {
		return contractConfigResponse{}, err
	}

	var data contractConfigResponse
	if err := cwClient.querySmartContractState(
		ctx,
		queryData,
		&data); err != nil {
		return contractConfigResponse{}, err
	}

	return data, nil
}

func (cwClient *CosmWasmClient) QueryConsumerId(ctx context.Context) (string, error) {
	data, err := cwClient.QueryConfig(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to query config")
	}

	return data.ConsumerId, nil
}

func (cwClient *CosmWasmClient) QueryFirstPubRandCommit(ctx context.Context, btcPkHex string) (*PubRandCommit, error) {
	queryData, err := newQueryFirstPubRandCommitMsg(btcPkHex)
	if err != nil {
		return nil, errors.Wrap(err, "newQueryFirstPubRandCommitMsg failed")
	}

	var commit PubRandCommit
	if err := cwClient.querySmartContractState(
		ctx,
		queryData,
		&commit); err != nil {
		return nil, errors.Wrap(err, "failed to QueryFirstPubRandCommit")
	}

	return &commit, nil
}

func (cwClient *CosmWasmClient) QueryLastPubRandCommit(ctx context.Context, btcPkHex string) (*PubRandCommit, error) {
	queryData, err := newQueryLastPubRandCommitMsg(btcPkHex)
	if err != nil {
		return nil, errors.Wrap(err, "newQueryLastPubRandCommitMsg failed")
	}

	var commit PubRandCommit
	if err := cwClient.querySmartContractState(
		ctx,
		queryData,
		&commit); err != nil {
		return nil, errors.Wrap(err, "failed to QueryLastPubRandCommit")
	}

	return &commit, nil
}

func (cwClient *CosmWasmClient) QueryIsEnabled(ctx context.Context) (bool, error) {
	queryData, err := newQueryIsEnabledMsg()
	if err != nil {
		return false, errors.Wrap(err, "newQueryIsEnabledMsg failed")
	}

	var isEnabled bool
	if err := cwClient.querySmartContractState(
		ctx,
		queryData,
		&isEnabled); err != nil {
		return false, errors.Wrap(err, "failed to query is enabled")
	}

	return isEnabled, nil
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
