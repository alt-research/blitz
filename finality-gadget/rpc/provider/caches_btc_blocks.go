package provider

import (
	"sort"
	"time"

	"github.com/babylonlabs-io/finality-gadget/types"
	"go.uber.org/zap"
)

type allPowerByBtcBlock struct {
	height          uint32    /// The height for this btc block
	latestFetchTime uint64    /// The latest time which fetch btc chain and got this block
	babylonBlock    [2]uint64 /// The babylon height interval covered by this btc block
}

func (b *allPowerByBtcBlock) isCoverBabylonBlock(block types.Block, newest bool) bool {
	if b.babylonBlock[0] <= block.BlockHeight && block.BlockHeight <= b.babylonBlock[1] {
		return true
	}

	if newest && block.BlockHeight > b.babylonBlock[1] {
		// make more 8s check for make sure covered
		if block.BlockTimestamp+8 < b.latestFetchTime {
			return true
		}
	}

	return false
}

func (b *allPowerByBtcBlock) onNewBabylonBlock(height uint64) {
	if b.babylonBlock[0] == 0 {
		b.babylonBlock = [2]uint64{height, height}
	}

	if height < b.babylonBlock[0] {
		b.babylonBlock[0] = height
	}

	if height > b.babylonBlock[1] {
		b.babylonBlock[1] = height
	}
}

type uint32Slice []uint32

func (x uint32Slice) Len() int           { return len(x) }
func (x uint32Slice) Less(i, j int) bool { return x[i] < x[j] }
func (x uint32Slice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

type allPowerByBtcBlocks struct {
	logger     *zap.SugaredLogger
	infos      map[uint32]*allPowerByBtcBlock
	btcHeights uint32Slice
	newest     uint32
}

func newAllPowerByBtcBlocks(logger *zap.SugaredLogger) *allPowerByBtcBlocks {
	return &allPowerByBtcBlocks{
		logger:     logger,
		infos:      make(map[uint32]*allPowerByBtcBlock, CacheMapCount),
		btcHeights: make([]uint32, 0, CacheMapCount),
	}
}

func (b *allPowerByBtcBlocks) insertNew(n *allPowerByBtcBlock) {
	b.infos[n.height] = n
	b.btcHeights = append(b.btcHeights, n.height)
	if n.height > b.newest {
		if n.height != b.newest+1 {
			b.logger.Debugw("skip a btc block", "from", b.newest, "to", n.height)
		}

		b.newest = n.height
	} else {
		// need sort
		sort.Sort(b.btcHeights)
	}
}

func (b *allPowerByBtcBlocks) onNewCache(babylonBlock *types.Block, btcHeight uint32) {
	b.logger.Debugw("on new cache", "btc", btcHeight, "babylon", babylonBlock.BlockHeight)

	curr, ok := b.infos[btcHeight]
	if !ok {
		n := &allPowerByBtcBlock{
			height:       btcHeight,
			babylonBlock: [2]uint64{babylonBlock.BlockHeight, babylonBlock.BlockHeight},
		}

		b.insertNew(n)
	}

	if curr != nil {
		curr.onNewBabylonBlock(babylonBlock.BlockHeight)
	}
}

func (b *allPowerByBtcBlocks) onNewBtcBlock(btcHeight uint32) {
	b.logger.Debugw("on new btc block", "btc", btcHeight)

	curr, ok := b.infos[btcHeight]
	if !ok {
		n := &allPowerByBtcBlock{
			height:          btcHeight,
			latestFetchTime: uint64(time.Now().Unix()),
		}

		b.insertNew(n)
	}

	if curr != nil {
		curr.latestFetchTime = uint64(time.Now().Unix())
	}
}

func (b *allPowerByBtcBlocks) logStatus() {
	b.logger.Debugw("btc blocks infos cache", "size", len(b.btcHeights), "newest", b.newest)

	if len(b.btcHeights) > 0 {
		start := len(b.btcHeights) - 1
		end := 0

		if len(b.btcHeights) >= 64 {
			end = len(b.btcHeights) - 64
		}

		for i := start; i >= end; i-- {
			height := b.btcHeights[i]
			info, ok := b.infos[height]
			if ok && info != nil {
				b.logger.Debugw("btc",
					"height", info.height,
					"from", info.babylonBlock[0],
					"to", info.babylonBlock[1],
					"latest", info.latestFetchTime)
			} else {
				b.logger.Warnw("no found btc info by height", "height", height)
			}
		}

		if len(b.btcHeights) > 64 {
			height := b.btcHeights[0]
			info, ok := b.infos[height]
			if ok && info != nil {
				b.logger.Debugw("btc",
					"height", info.height,
					"from", info.babylonBlock[0],
					"to", info.babylonBlock[1],
					"latest", info.latestFetchTime)
			} else {
				b.logger.Warnw("no found btc info by height", "height", height)
			}
		}
	}
}
