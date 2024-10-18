package finalityprovider

import (
	"context"
	"time"
)

func (fp *FinalityProvider) Start(ctx context.Context) {
	fp.wg.Add(1)

	go func() {
		defer func() {
			fp.logger.Info("Stop l2 block handler")
			fp.wg.Done()
		}()

		fp.logger.Info("Starting l2 block handler")

		ticker := time.NewTicker(fp.tickInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case blk := <-fp.l2BlocksChan:
				fp.logger.Debug("on new l2 block", "number", blk.NumberU64())
				fp.committer.Append(blk)
				err := fp.committer.TryCommitPendingBlock(ctx)
				if err != nil {
					fp.logger.Error("tryCommitPendingBlock failed", "err", err)
				}
			case <-ticker.C:
				fp.logger.Debug("on finality provider ticker")
				err := fp.tick(ctx)
				if err != nil {
					fp.logger.Error("fetch l2 block handler error", "err", err)
				}
			}
		}
	}()
}

func (fp *FinalityProvider) tick(ctx context.Context) error {
	isEnable, err := fp.cwClient.QueryIsEnabled(ctx)
	if err != nil {
		fp.logger.Error("QueryIsEnabled failed", "err", err)
	}

	config, err := fp.cwClient.QueryConfig(ctx)
	if err != nil {
		fp.logger.Error("QueryConfig failed", "err", err)
	}

	fp.logger.Debug("cw", "isEnable", isEnable, "config", config)

	err = fp.committer.TryCommitPendingBlock(ctx)
	if err != nil {
		fp.logger.Error("tryCommitPendingBlock on tick failed", "err", err)
	}

	return nil
}

func (fp *FinalityProvider) Wait() {
	fp.wg.Wait()
}
