package provider

import (
	"strings"
	"time"

	"github.com/babylonlabs-io/finality-gadget/types"
	"github.com/pkg/errors"
)

func (p *FinalizedStateProvider) queryAllFpBtcPubKeys() ([]string, error) {
	res, useCache := func() ([]string, bool) {
		p.cacheMu.RLock()
		defer p.cacheMu.RUnlock()

		p.logger.Sugar().Debugw("check cache for all fp", "cache len", len(p.allFpsCache), "current", time.Now(), "cache", p.allFpsCacheLastTime)
		if len(p.allFpsCache) > 0 && isTimeNotGreaterThan(p.allFpsCacheLastTime) {
			return p.allFpsCache, true
		}

		return nil, false
	}()

	if useCache {
		p.logger.Sugar().Debugw("use cache for all fp btc keys", "res", res)
		return res, nil
	}

	// get the consumer chain id
	consumerId, err := p.cwClient.QueryConsumerId()
	if err != nil {
		return nil, err
	}

	// get all the FPs pubkey for the consumer chain
	allFpPks, err := p.bbnClient.QueryAllFpBtcPubKeys(consumerId)
	if err != nil {
		return nil, err
	}

	func() {
		p.cacheMu.Lock()
		defer p.cacheMu.Unlock()

		p.allFpsCache = allFpPks
		p.allFpsCacheLastTime = time.Now()
	}()

	return allFpPks, nil
}

func (p *FinalizedStateProvider) queryListOfVotedFinalityProviders(queryParams *types.Block) ([]string, error) {
	res, useCache := func() ([]string, bool) {
		p.cacheMu.RLock()
		defer p.cacheMu.RUnlock()

		c, ok := p.votedFpPksCache[queryParams.BlockHash]
		if ok {
			return c, true
		}

		return nil, false
	}()

	if useCache {
		return res, nil
	}

	votedFpPks, err := p.cwClient.QueryListOfVotedFinalityProviders(queryParams)

	if err == nil {
		func() {
			p.cacheMu.Lock()
			defer p.cacheMu.Unlock()

			if len(p.votedFpPksCache) > CacheMapCount {
				p.votedFpPksCache = make(map[string][]string, CacheMapCount)
			}

			p.votedFpPksCache[queryParams.BlockHash] = votedFpPks
		}()
	}

	if len(votedFpPks) == 0 {
		p.logger.Sugar().Debugw("not found voted finality provider", "block", queryParams.BlockHeight)
	}

	return votedFpPks, err
}

func (p *FinalizedStateProvider) queryBlockHeightByTimestamp(block *types.Block) (uint32, error) {
	// convert the L2 timestamp to BTC height
	btcblockHeight, err := p.btcClient.GetBlockHeightByTimestamp(block.BlockTimestamp)
	if err != nil {
		return 0, errors.Wrap(err, "GetBlockHeightByTimestamp")
	}

	return btcblockHeight, err

}

func (p *FinalizedStateProvider) queryEarliestActiveDelBtcHeight(fpPubkeyHexList []string) (uint32, error) {
	key := strings.Join(fpPubkeyHexList, ",")

	res, useCache := func() (uint32, bool) {
		p.cacheMu.RLock()
		defer p.cacheMu.RUnlock()

		c, ok := p.earliestActiveDelBtcHeightCache[key]
		if ok {
			return c, true
		}

		return 0, false
	}()

	if useCache {
		return res, nil
	}

	// check whether the btc staking is actived
	earliestDelHeight, err := p.bbnClient.QueryEarliestActiveDelBtcHeight(fpPubkeyHexList)
	if err != nil {
		return 0, errors.Wrap(err, "QueryEarliestActiveDelBtcHeight")
	}

	if err == nil {
		func() {
			p.cacheMu.Lock()
			defer p.cacheMu.Unlock()

			if len(p.earliestActiveDelBtcHeightCache) > CacheMapCount {
				p.earliestActiveDelBtcHeightCache = make(map[string]uint32, CacheMapCount)
			}

			p.earliestActiveDelBtcHeightCache[key] = earliestDelHeight
		}()
	}

	return earliestDelHeight, err
}

func (p *FinalizedStateProvider) queryMultiFpPower(fpPubkeyHexList []string, btcHeight uint32) (map[string]uint64, error) {
	res, useCache := func() (map[string]uint64, bool) {
		p.cacheMu.RLock()
		defer p.cacheMu.RUnlock()

		c, ok := p.multiFpPowerCache[btcHeight]
		if ok {
			return c, true
		}

		return nil, false
	}()

	if useCache {
		return res, nil
	}

	// get all FPs voting power at this BTC height
	allFpPower, err := p.bbnClient.QueryMultiFpPower(fpPubkeyHexList, btcHeight)
	if err != nil {
		return nil, errors.Wrap(err, "QueryMultiFpPower")
	}

	if err == nil {
		func() {
			p.cacheMu.Lock()
			defer p.cacheMu.Unlock()

			if len(p.multiFpPowerCache) > CacheMapCount {
				p.multiFpPowerCache = make(map[uint32]map[string]uint64, CacheMapCount)
			}

			p.multiFpPowerCache[btcHeight] = allFpPower
		}()
	}

	return allFpPower, err
}

func (p *FinalizedStateProvider) queryAllPkPower(block *types.Block) (map[string]uint64, error) {
	p.logger.Sugar().Debugw("queryAllPkPower", "block", block.BlockTimestamp, "current", time.Now().Unix())

	// get all FPs pubkey for the consumer chain
	allFpPks, err := p.allFpPksCache.QueryAllFpBtcPubKeys()
	if err != nil {
		return nil, errors.Wrap(err, "queryAllFpBtcPubKeys")
	}

	p.logger.Sugar().Infof("allFpPks %v", allFpPks)

	// convert the L2 timestamp to BTC height
	btcblockHeight, err := p.queryBlockHeightByTimestamp(block)
	if err != nil {
		return nil, errors.Wrap(err, "queryBlockHeightByTimestamp")
	}

	p.logger.Sugar().Infof("btcblockHeight %v", btcblockHeight)

	// check whether the btc staking is actived
	// earliestDelHeight, err := p.queryEarliestActiveDelBtcHeight(allFpPks)
	// if err != nil {
	//	return nil, errors.Wrap(err, "QueryEarliestActiveDelBtcHeight")
	//}

	// p.logger.Sugar().Info("earliestDelHeight ", earliestDelHeight)

	// if btcblockHeight < earliestDelHeight {
	//   return nil, errors.Wrapf(types.ErrBtcStakingNotActivated, "current %v, earliest %v", btcblockHeight, earliestDelHeight)
	// }

	// get all FPs voting power at this BTC height
	allFpPower, err := p.queryMultiFpPower(allFpPks, btcblockHeight)
	if err != nil {
		return nil, errors.Wrap(err, "QueryMultiFpPower")
	}

	p.logger.Sugar().Info("allFpPower ", allFpPower)

	p.fpPowerCache.SetCache(block, btcblockHeight, allFpPower)
	p.fpPowerCache.LogCacheStatus()

	return allFpPower, nil
}
