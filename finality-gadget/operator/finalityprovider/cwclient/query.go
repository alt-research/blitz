package cwclient

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

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
