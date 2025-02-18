package controllers

import (
	"context"
	"encoding/json"
	"math/big"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	bbntypes "github.com/babylonlabs-io/babylon/types"
	"github.com/babylonlabs-io/finality-gadget/db"
	"github.com/babylonlabs-io/finality-provider/clientcontroller/api"
	"github.com/babylonlabs-io/finality-provider/clientcontroller/opstackl2"
	cwcclient "github.com/babylonlabs-io/finality-provider/cosmwasmclient/client"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/babylonlabs-io/finality-provider/types"

	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
	"github.com/alt-research/blitz/finality-gadget/operator/configs"
)

var _ api.ConsumerController = &OrbitConsumerController{}

type OrbitConsumerController struct {
	cfg    *fpcfg.OPStackL2Config
	logger *zap.Logger

	ctx      context.Context
	cwClient *cwcclient.Client
	l2Client *l2eth.L2EthClient

	activeHeight    uint64
	backHeightCount uint64
}

func NewOrbitConsumerController(
	ctx context.Context,
	cfg *configs.OperatorConfig,
	fpConfig *fpcfg.OPStackL2Config,
	zapLogger *zap.Logger,
) (*OrbitConsumerController, error) {
	if err := fpConfig.Validate(); err != nil {
		return nil, err
	}
	cwConfig := fpConfig.ToCosmwasmConfig()

	cwClient, err := opstackl2.NewCwClient(&cwConfig, zapLogger)
	if err != nil {
		return nil, errors.Errorf("failed to create CW client: %w", err)
	}

	l2Client, err := l2eth.NewL2EthClient(ctx, &cfg.Layer2)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create l2 eth client")
	}

	// Init local DB for storing and querying blocks
	db, err := db.NewBBoltHandler(cfg.Babylon.FinalityGadgetCfg.DBFilePath, zapLogger)
	if err != nil {
		return nil, errors.Errorf("failed to create DB handler: %w", err)
	}
	defer db.Close()
	err = db.CreateInitialSchema()
	if err != nil {
		return nil, errors.Errorf("create initial buckets error: %w", err)
	}

	return &OrbitConsumerController{
		cfg:             fpConfig,
		logger:          zapLogger,
		ctx:             ctx,
		cwClient:        cwClient,
		l2Client:        l2Client,
		activeHeight:    cfg.Layer2.ActivatedHeight,
		backHeightCount: cfg.Layer2.BackHeightCount,
	}, nil
}

func (wc *OrbitConsumerController) Ctx() context.Context {
	return wc.ctx
}

// CommitPubRandList commits a list of EOTS public randomness the consumer chain
// it returns tx hash and error
func (wc *OrbitConsumerController) CommitPubRandList(
	fpPk *btcec.PublicKey,
	startHeight uint64,
	numPubRand uint64,
	commitment []byte,
	sig *schnorr.Signature) (*types.TxResponse, error) {
	wc.logger.Sugar().Debugf("CommitPubRandList %v %v", startHeight, wc.cwClient.MustGetAddr())
	msg := opstackl2.CommitPublicRandomnessMsg{
		CommitPublicRandomness: opstackl2.CommitPublicRandomnessMsgParams{
			FpPubkeyHex: bbntypes.NewBIP340PubKeyFromBTCPK(fpPk).MarshalHex(),
			StartHeight: startHeight,
			NumPubRand:  numPubRand,
			Commitment:  commitment,
			Signature:   sig.Serialize(),
		},
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	execMsg := &wasmtypes.MsgExecuteContract{
		Sender:   wc.cwClient.MustGetAddr(),
		Contract: wc.cfg.OPFinalityGadgetAddress,
		Msg:      payload,
	}

	res, err := wc.cwClient.ReliablySendMsg(wc.Ctx(), execMsg, nil, nil)
	if err != nil {
		return nil, err
	}
	return &types.TxResponse{TxHash: res.TxHash}, nil
}

// SubmitBatchFinalitySigs submits a batch of finality signatures to the consumer chain
func (wc *OrbitConsumerController) SubmitBatchFinalitySigs(
	fpPk *btcec.PublicKey,
	blocks []*types.BlockInfo,
	pubRandList []*btcec.FieldVal,
	proofList [][]byte,
	sigs []*btcec.ModNScalar) (*types.TxResponse, error) {
	wc.logger.Sugar().Debugf("SubmitBatchFinalitySigs %v", blocks)
	tx, err := wc.submitBatchFinalitySigs(fpPk, blocks, pubRandList, proofList, sigs)
	if err != nil {
		wc.logger.Sugar().Errorf("SubmitFinalitySig %v failed: %v", blocks, err)
	}
	return tx, err
}

// Note: the following queries are only for PoC

// QueryFinalityProviderHasPower queries whether the finality provider has voting power at a given height
func (wc *OrbitConsumerController) QueryFinalityProviderHasPower(fpPk *btcec.PublicKey, blockHeight uint64) (bool, error) {
	//return wc.inner.QueryFinalityProviderHasPower(fpPk, blockHeight)
	return true, nil
}

// QueryLatestFinalizedBlock returns the latest finalized block
// Note: nil will be returned if the finalized block does not exist
func (wc *OrbitConsumerController) QueryLatestFinalizedBlock() (*types.BlockInfo, error) {
	logger := wc.logger.Sugar()
	logger.Debugf("QueryLatestFinalizedBlock")

	res, err := wc.queryBlock(rpc.BlockNumberOrHashWithNumber(rpc.FinalizedBlockNumber))

	if err != nil {
		logger.Errorf("QueryLatestFinalizedBlock failed by %v", err)
	} else {
		logger.Debugf("QueryLatestFinalizedBlock res %v", res)
	}

	return res, err
}

// QueryLastPublicRandCommit returns the last committed public randomness
func (wc *OrbitConsumerController) QueryLastPublicRandCommit(fpPk *btcec.PublicKey) (*types.PubRandCommit, error) {
	fpPubKey := bbntypes.NewBIP340PubKeyFromBTCPK(fpPk)
	queryMsg := &opstackl2.QueryMsg{
		LastPubRandCommit: &opstackl2.PubRandCommit{
			BtcPkHex: fpPubKey.MarshalHex(),
		},
	}

	jsonData, err := json.Marshal(queryMsg)
	if err != nil {
		return nil, errors.Errorf("failed marshaling to JSON: %w", err)
	}

	stateResp, err := wc.cwClient.QuerySmartContractState(context.Background(), wc.cfg.OPFinalityGadgetAddress, string(jsonData))
	if err != nil {
		return nil, errors.Errorf("failed to query smart contract state: %w", err)
	}
	if len(stateResp.Data) == 0 {
		return nil, nil
	}

	var resp *types.PubRandCommit
	err = json.Unmarshal(stateResp.Data, &resp)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal response: %w", err)
	}
	if resp == nil {
		return nil, nil
	}
	if err := resp.Validate(); err != nil {
		return nil, err
	}

	return resp, nil
}

// QueryBlock queries the block at the given height
func (wc *OrbitConsumerController) QueryBlock(height uint64) (*types.BlockInfo, error) {
	res, err := wc.QueryBlocks(height, height, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query block by %v", height)
	}

	if len(res) != 1 {
		return nil, errors.Errorf("query blocks returned no block information for %v", height)
	}

	return res[0], nil
}

// QueryIsBlockFinalized queries if the block at the given height is finalized
func (wc *OrbitConsumerController) QueryIsBlockFinalized(height uint64) (bool, error) {
	l2Block, err := wc.QueryLatestFinalizedBlock()
	if err != nil {
		return false, err
	}

	if l2Block == nil {
		return false, nil
	}
	if height > l2Block.Height {
		return false, nil
	}
	return true, nil
}

// QueryBlocks returns a list of blocks from startHeight to endHeight
func (wc *OrbitConsumerController) QueryBlocks(startHeight, endHeight uint64, limit uint32) ([]*types.BlockInfo, error) {
	if endHeight < startHeight {
		// no need return error
		return nil, nil
	}
	count := endHeight - startHeight + 1
	if count > uint64(limit) {
		count = uint64(limit)
	}

	if count == 0 {
		wc.logger.Warn("QueryBlocks count is zero!")
		return nil, nil
	}

	res := make([]*types.BlockInfo, 0, count)
	for i := uint64(0); i < count; i++ {
		h := big.NewInt(int64(startHeight + i))
		header, err := wc.l2Client.HeaderByNumber(wc.Ctx(), h)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get header by number by %v", h)
		}

		hash := header.Hash()

		res = append(res, &types.BlockInfo{
			Height: h.Uint64(),
			Hash:   hash[:],
		})
	}

	return res, nil
}

// QueryLatestBlockHeight queries the tip block height of the consumer chain
func (wc *OrbitConsumerController) QueryLatestBlockHeight() (uint64, error) {
	logger := wc.logger.Sugar()
	// logger.Debugf("QueryLatestBlockHeight")

	res, err := wc.queryBlock(rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
	if err != nil {
		logger.Errorf("QueryLatestBlockHeight failed by %v", err)
		return 0, err
	}

	height := res.Height
	if height <= wc.backHeightCount {
		height = 1
	} else {
		height = height - wc.backHeightCount
	}

	// logger.Debugf("QueryLatestBlockHeight res %v", height)

	return height, err
}

// QueryActivatedHeight returns the activated height of the consumer chain
// error will be returned if the consumer chain has not been activated
func (wc *OrbitConsumerController) QueryActivatedHeight() (uint64, error) {
	return wc.activeHeight, nil
}

// QueryFinalityProviderSlashedOrJailed - returns if the fp has been slashed, jailed, err
// nolint:revive // Ignore stutter warning - full name provides clarity
func (cc *OrbitConsumerController) QueryFinalityProviderSlashedOrJailed(fpPk *btcec.PublicKey) (bool, bool, error) {
	// TODO: implement slashed or jailed feature in OP stack L2
	return false, false, nil
}

// QueryFinalityActivationBlockHeight returns the block height of the consumer chain
// starts to accept finality voting and pub rand commit as start height
// error will be returned if the consumer chain failed to get this value
// if the consumer chain wants to accept finality voting at any block height
// the value zero should be returned.
func (wc *OrbitConsumerController) QueryFinalityActivationBlockHeight() (uint64, error) {
	// TODO: implement finality activation feature in OP stack L2
	return 0, nil
}

// nolint:revive // Ignore stutter warning - full name provides clarity
func (cc *OrbitConsumerController) QueryFinalityProviderHighestVotedHeight(fpPk *btcec.PublicKey) (uint64, error) {
	// TODO: implement highest voted height feature in OP stack L2
	return 0, nil
}

// nolint:revive // Ignore stutter warning - full name provides clarity
func (cc *OrbitConsumerController) UnjailFinalityProvider(fpPk *btcec.PublicKey) (*types.TxResponse, error) {
	// TODO: implement unjail feature in OP stack L2
	return nil, nil
}

func (wc *OrbitConsumerController) Close() error {
	wc.l2Client.Close()
	return wc.cwClient.Stop()
}
