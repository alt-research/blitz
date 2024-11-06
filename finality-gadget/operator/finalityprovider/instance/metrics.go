package fp_instance

func (fp *FinalityProviderInstance) recordOrbitFinalizedHeight() {
	latestFinalizedBlock, err := fp.consumerCon.QueryLatestFinalizedBlock()
	if err != nil {
		fp.logger.Sugar().Errorf("query latest finalized height failed by %v", err.Error())
		return
	}

	if latestFinalizedBlock.Height != 0 {
		fp.logger.Sugar().Infof("RecordOrbitFinalizedHeight %v", latestFinalizedBlock.Height)
		fp.blitzMetrics.RecordOrbitFinalizedHeight("", latestFinalizedBlock.Height)
	}
}
