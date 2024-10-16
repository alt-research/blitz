package finalityprovider

import (
	"github.com/ethereum/go-ethereum/core/types"

	fgtypes "github.com/babylonlabs-io/finality-gadget/types"
)

func ToFgBlock(blk *types.Block) *fgtypes.Block {
	return &fgtypes.Block{
		BlockHash:      blk.Hash().Hex(),
		BlockHeight:    blk.NumberU64(),
		BlockTimestamp: blk.Time(),
	}
}
