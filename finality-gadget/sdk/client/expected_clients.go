package finalitygadget

import (
	"context"

	"github.com/babylonlabs-io/finality-gadget/types"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

type IBitcoinClient interface {
	GetBlockCount() (uint64, error)
	GetBlockHashByHeight(height uint64) (*chainhash.Hash, error)
	GetBlockHeaderByHash(blockHash *chainhash.Hash) (*wire.BlockHeader, error)
	GetBlockHeightByTimestamp(targetTimestamp uint64) (uint64, error)
	GetBlockTimestampByHeight(height uint64) (uint64, error)
}

type IBabylonClient interface {
	QueryAllFpBtcPubKeys(consumerId string) ([]string, error)
	QueryFpPower(fpPubkeyHex string, btcHeight uint64) (uint64, error)
	QueryMultiFpPower(fpPubkeyHexList []string, btcHeight uint64) (map[string]uint64, error)
	QueryEarliestActiveDelBtcHeight(fpPubkeyHexList []string) (uint64, error)
}

type ICosmWasmClient interface {
	QueryListOfVotedFinalityProviders(ctx context.Context, queryParams *types.Block) ([]string, error)
	QueryConsumerId(ctx context.Context) (string, error)
	QueryIsEnabled(ctx context.Context) (bool, error)
}
