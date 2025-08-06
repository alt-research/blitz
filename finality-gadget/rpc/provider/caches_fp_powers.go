package provider

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/babylonlabs-io/finality-gadget/finalitygadget"
	"github.com/babylonlabs-io/finality-gadget/types"
	"go.uber.org/zap"
)

type FpPowerCacheByHeight struct {
	powers         map[string]uint64
	cacheTime      uint64
	btcBlockNumber uint32
}

type FpPowerCaches struct {
	logger *zap.SugaredLogger

	caches       []*FpPowerCacheByHeight
	blocksCached map[uint32]*FpPowerCacheByHeight
	cachesByBtc  *allPowerByBtcBlocks

	btcClient          finalitygadget.IBitcoinClient
	fetchBlockInterval time.Duration
	cacheMu            sync.RWMutex
}

func NewFpPowerCaches(
	ctx context.Context,
	logger *zap.Logger,
	btcClient finalitygadget.IBitcoinClient,
	fetchBlockInterval time.Duration) *FpPowerCaches {
	res := &FpPowerCaches{
		logger:             logger.Sugar(),
		caches:             make([]*FpPowerCacheByHeight, 0, CacheMapCount),
		btcClient:          btcClient,
		fetchBlockInterval: fetchBlockInterval,
		blocksCached:       make(map[uint32]*FpPowerCacheByHeight, CacheMapCount),
		cachesByBtc:        newAllPowerByBtcBlocks(logger.Sugar()),
	}

	go func() {
		res.serv(ctx)
	}()

	return res
}

func (f *FpPowerCaches) Len() int {
	return len(f.caches)
}

func (f *FpPowerCaches) Less(i, j int) bool {
	return f.caches[i].btcBlockNumber < f.caches[j].btcBlockNumber
}

func (f *FpPowerCaches) Swap(i, j int) {
	tmp := f.caches[i]
	f.caches[i] = f.caches[j]
	f.caches[j] = tmp
}

func (f *FpPowerCaches) SetCache(block *types.Block, btcBlockNumber uint32, powers map[string]uint64) {
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()

	cacheBlk, cached := f.blocksCached[btcBlockNumber]
	if cached {
		if cacheBlk.btcBlockNumber == math.MaxUint32 {
			cacheBlk.btcBlockNumber = btcBlockNumber
		}
		return
	}

	f.logger.Debugw("cache new fp power", "block", block, "powers", powers)

	n := &FpPowerCacheByHeight{
		powers:         powers,
		cacheTime:      uint64(time.Now().Unix()),
		btcBlockNumber: btcBlockNumber,
	}
	f.caches = append(f.caches, n)
	f.blocksCached[btcBlockNumber] = n

	if len(f.caches) > CacheMapCount {
		// clean old caches
		caches := make([]*FpPowerCacheByHeight, 0, CacheMapCount)
		blocksCached := make(map[uint32]*FpPowerCacheByHeight, CacheMapCount)

		for i := CleanCacheCount; i < len(f.caches); i++ {
			caches = append(caches, f.caches[i])
			blocksCached[f.caches[i].btcBlockNumber] = f.caches[i]
		}

		f.caches = caches
		f.blocksCached = blocksCached
	}

	sort.Stable(f)

	f.cachesByBtc.onNewCache(block, btcBlockNumber)
}

func (f *FpPowerCaches) FindCache(block *types.Block) (map[string]uint64, bool) {
	f.cacheMu.RLock()
	defer f.cacheMu.RUnlock()

	return nil, false
}

func (f *FpPowerCaches) LogCacheStatus() {
	f.cacheMu.RLock()
	defer f.cacheMu.RUnlock()

	f.logger.Debugw("fp power cache", "size", len(f.blocksCached), "array", len(f.caches))
	for i := 0; i < len(f.caches); i++ {
		f.logger.Debugw("cache", "idx", i, "btc", f.caches[i].btcBlockNumber, "allpower", f.caches[i].powers)
	}

	f.cachesByBtc.logStatus()

}

func (f *FpPowerCaches) serv(ctx context.Context) {
	defer func() {
		f.logger.Info("Stop fp all power cache btc block handler")
	}()

	f.logger.Info("Starting l2 block handler")

	ticker := time.NewTicker(f.fetchBlockInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.logger.Debug("on block handler ticker")
			err := f.tryFetchBtcBlock(ctx)
			if err != nil {
				f.logger.Errorf("fp all power fetch btc block failed: %w", err)
			}
		}
	}
}

func (f *FpPowerCaches) tryFetchBtcBlock(ctx context.Context) error {
	blockHeight, err := f.btcClient.GetBlockCount()
	if err != nil {
		return fmt.Errorf("get block count failed: %w", err)
	}

	func() {
		f.cacheMu.Lock()
		defer f.cacheMu.Unlock()

		f.cachesByBtc.onNewBtcBlock(blockHeight)
	}()

	return nil
}
