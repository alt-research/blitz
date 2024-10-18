package finalityprovider

import (
	"time"

	"github.com/avast/retry-go/v4"
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

var (
	// TODO: Maybe configurable?
	RtyAttNum = uint(5)
	RtyAtt    = retry.Attempts(RtyAttNum)
	RtyDel    = retry.Delay(time.Millisecond * 400)
	RtyErr    = retry.LastErrorOnly(true)
)
