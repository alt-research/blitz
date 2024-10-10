package cwclient

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

type ICosmosWasmContractClient interface {
	QueryListOfVotedFinalityProviders(
		ctx context.Context,
		height uint64,
		hash common.Hash,
	) ([]string, error)

	QueryConfig(ctx context.Context) (contractConfigResponse, error)

	QueryConsumerId(ctx context.Context) (string, error)

	QueryFirstPubRandCommit(ctx context.Context, btcPkHex string) (*PubRandCommit, error)

	QueryLastPubRandCommit(ctx context.Context, btcPkHex string) (*PubRandCommit, error)

	QueryIsEnabled(ctx context.Context) (bool, error)
}
