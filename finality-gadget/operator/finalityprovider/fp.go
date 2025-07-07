package finalityprovider

import (
	"context"
	"fmt"
	"strings"

	bbntypes "github.com/babylonlabs-io/babylon/types"
	bstypes "github.com/babylonlabs-io/babylon/x/btcstaking/types"
	"github.com/babylonlabs-io/finality-provider/finality-provider/proto"
	"github.com/babylonlabs-io/finality-provider/finality-provider/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (app *FinalityProviderApp) RestoreFP(ctx context.Context, keyName, chainID string, fpBtpPkStr string) error {
	fpBtpPk, err := bbntypes.NewBIP340PubKeyFromHex(fpBtpPkStr)
	if err != nil {
		return errors.Wrapf(err, "failed to NewBIP340PubKeyFromHex by %v", fpBtpPk)
	}

	// Query the consumer chain to check if the fp is already registered
	// if true, update db with the fp info from the consumer chain
	// otherwise, proceed registration
	resp, err := app.fpApp.GetBabylonController().QueryFinalityProvider(fpBtpPk.MustToBTCPK())
	if err != nil {
		if !strings.Contains(err.Error(), "the finality provider is not found") {
			return fmt.Errorf("err getting finality provider: %w", err)
		}

		return errors.Wrapf(err, "query finality provider %s failed", fpBtpPkStr)
	}

	if resp == nil {
		return errors.Errorf("no found finality provider by %s", fpBtpPkStr)
	}

	if err := app.putFpFromResponse(resp.FinalityProvider, chainID); err != nil {
		return errors.Wrap(err, "putFpFromResponse failed")
	}

	// get updated fp from db
	_, err = app.fpApp.GetFinalityProviderStore().GetFinalityProvider(fpBtpPk.MustToBTCPK())
	if err != nil {
		return errors.Wrap(err, "GetFinalityProvider failed")
	}

	return nil
}

// putFpFromResponse creates or updates finality-provider in the local store
func (app *FinalityProviderApp) putFpFromResponse(fp *bstypes.FinalityProviderResponse, chainID string) error {
	btcPk := fp.BtcPk.MustToBTCPK()
	_, err := app.fpApp.GetFinalityProviderStore().GetFinalityProvider(btcPk)
	if err != nil {
		if errors.Is(err, store.ErrFinalityProviderNotFound) {
			addr, err := sdk.AccAddressFromBech32(fp.Addr)
			if err != nil {
				return fmt.Errorf("err converting fp addr: %w", err)
			}

			if fp.Commission == nil {
				return errors.New("nil Commission in FinalityProviderResponse")
			}

			if fp.CommissionInfo == nil {
				return errors.New("nil CommissionInfo in FinalityProviderResponse")
			}

			commRates := bstypes.NewCommissionRates(*fp.Commission, fp.CommissionInfo.MaxRate, fp.CommissionInfo.MaxChangeRate)

			if err := app.fpApp.GetFinalityProviderStore().CreateFinalityProvider(addr, btcPk, fp.Description, commRates, chainID); err != nil {
				return fmt.Errorf("failed to save finality-provider: %w", err)
			}

			app.logger.Info("finality-provider successfully saved the local db",
				zap.String("eots_pk", fp.BtcPk.MarshalHex()),
				zap.String("addr", fp.Addr),
			)

			return nil
		}

		return err
	}

	if err := app.fpApp.GetFinalityProviderStore().SetFpDescription(btcPk, fp.Description, fp.Commission); err != nil {
		return err
	}

	if err := app.fpApp.GetFinalityProviderStore().SetFpLastVotedHeight(btcPk, uint64(fp.HighestVotedHeight)); err != nil {
		return err
	}

	hasPower, err := app.fpApp.GetConsumerController().QueryFinalityProviderHasPower(btcPk, fp.Height)
	if err != nil {
		return fmt.Errorf("failed to query voting power for finality provider %s: %w",
			fp.BtcPk.MarshalHex(), err)
	}

	var status proto.FinalityProviderStatus
	switch {
	case hasPower:
		status = proto.FinalityProviderStatus_ACTIVE
	case fp.SlashedBtcHeight > 0:
		status = proto.FinalityProviderStatus_SLASHED
	case fp.Jailed:
		status = proto.FinalityProviderStatus_JAILED
	default:
		status = proto.FinalityProviderStatus_INACTIVE
	}

	if err := app.fpApp.GetFinalityProviderStore().SetFpStatus(btcPk, status); err != nil {
		return fmt.Errorf("failed to update status for finality provider %s: %w", fp.BtcPk.MarshalHex(), err)
	}

	app.logger.Info("finality-provider successfully updated the local db",
		zap.String("eots_pk", fp.BtcPk.MarshalHex()),
		zap.String("addr", fp.Addr),
	)

	return nil
}
