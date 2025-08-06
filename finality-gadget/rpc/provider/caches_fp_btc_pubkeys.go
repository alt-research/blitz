package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/babylonlabs-io/finality-gadget/finalitygadget"
	"go.uber.org/zap"
)

// FpBtcPubKeysCache is the cache for fp btc pub keys by a babylon 's consumerId
// NOTE in babylon the fp btc pub keys for a consumer will never be delete,
// it just will have no powers to sign, so we can just make a set for all pub keys
// we just need make sure it can updates from babylon chain
type FpBtcPubKeysCache struct {
	fpBtcPubKeysMap map[string]bool
	fpBtcPubKeys    []string

	updateTime      time.Time
	timeoutDuration time.Duration
	fetchInterval   time.Duration

	consumerId string

	bbnClient finalitygadget.IBabylonClient

	logger *zap.SugaredLogger
	mu     sync.RWMutex
}

func NewFpBtcPubKeysCache(ctx context.Context,
	logger *zap.Logger,
	timeoutDuration time.Duration,
	fetchInterval time.Duration,
	bbnClient finalitygadget.IBabylonClient,
	cwClient finalitygadget.ICosmWasmClient) (*FpBtcPubKeysCache, error) {
	// get the consumer chain id
	consumerId, err := cwClient.QueryConsumerId()
	if err != nil {
		return nil, fmt.Errorf("try QueryConsumerId failed: %w", err)
	}

	res := &FpBtcPubKeysCache{
		fpBtcPubKeysMap: make(map[string]bool, 128),
		fpBtcPubKeys:    make([]string, 0, 128),
		timeoutDuration: timeoutDuration,
		fetchInterval:   fetchInterval,
		consumerId:      consumerId,
		bbnClient:       bbnClient,
		logger:          logger.Sugar(),
	}

	if err := res.tryFetch(ctx); err != nil {
		return nil, fmt.Errorf("try fetch in init failed: %w", err)
	}

	go func() {
		res.serv(ctx)
	}()

	return res, nil
}

func (f *FpBtcPubKeysCache) serv(ctx context.Context) {
	defer func() {
		f.logger.Info("Stop fp btc pub keys cache fetcher")
	}()

	f.logger.Info("Starting fp btc pub keys cache fetcher")

	ticker := time.NewTicker(f.fetchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.logger.Debug("on block handler ticker")
			err := f.tryFetch(ctx)
			if err != nil {
				f.logger.Errorf("fp all power fetch btc block failed: %w", err)
			}
		}
	}
}

func (f *FpBtcPubKeysCache) tryFetch(ctx context.Context) error {
	// get all the FPs pubkey for the consumer chain
	allFpPks, err := f.bbnClient.QueryAllFpBtcPubKeys(f.consumerId)
	if err != nil {
		return fmt.Errorf("failed to QueryAllFpBtcPubKeys: %w", err)
	}

	f.mergeIntoCache(allFpPks)

	return nil
}

func (f *FpBtcPubKeysCache) mergeIntoCache(pks []string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, fp := range pks {
		_, ok := f.fpBtcPubKeysMap[fp]
		if !ok {
			f.logger.Debugw("new fp pk in cache", "pk", fp)
			f.fpBtcPubKeys = append(f.fpBtcPubKeys, fp)
			f.fpBtcPubKeysMap[fp] = true
		}
	}

	f.updateTime = time.Now()
}

func (f *FpBtcPubKeysCache) QueryAllFpBtcPubKeys() ([]string, error) {
	res, isUseCache := func() ([]string, bool) {
		f.mu.RLock()
		defer f.mu.RUnlock()

		isUse := isUpdateTimeNoTimeout(f.updateTime, f.timeoutDuration)
		if isUse {
			f.logger.Debugw("use all fp btc pub keys caches", "pks", f.fpBtcPubKeys)
			resCache := make([]string, 0, len(f.fpBtcPubKeys))
			copy(resCache, f.fpBtcPubKeys)
			return resCache, true
		} else {
			return nil, false
		}
	}()

	if isUseCache {
		return res, nil
	}

	// get all the FPs pubkey for the consumer chain
	allFpPks, err := f.bbnClient.QueryAllFpBtcPubKeys(f.consumerId)
	if err != nil {
		return nil, fmt.Errorf("failed to QueryAllFpBtcPubKeys: %w", err)
	}

	f.mergeIntoCache(allFpPks)

	return allFpPks, nil
}
