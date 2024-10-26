package controllers

import (
	"github.com/babylonlabs-io/finality-provider/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

func (wc *OrbitConsumerController) queryBlock(number rpc.BlockNumberOrHash) (*types.BlockInfo, error) {
	blks, err := wc.l2Client.BlockReceipts(wc.Ctx(), number)
	if err != nil {
		return nil, errors.Wrap(err, "failed to QueryBestBlock by get SafeBlockNumber BlockReceipts")
	}

	if len(blks) == 0 {
		wc.logger.Warn("get QueryBestblock by get SafeBlockNumber no returns")
		return nil, nil
	}

	block := blks[0]
	blockHash := block.BlockHash

	return &types.BlockInfo{
		Height: block.BlockNumber.Uint64(),
		Hash:   blockHash[:],
	}, nil
}
