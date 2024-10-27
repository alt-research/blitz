package finalityprovider

import (
	"context"

	bbntypes "github.com/babylonlabs-io/babylon/types"
	"github.com/pkg/errors"
)

func (app *FinalityProviderApp) RestoreFP(ctx context.Context, keyName string, fpBtpPk string) error {
	_, err := bbntypes.NewBIP340PubKeyFromHex(fpBtpPk)
	if err != nil {
		return errors.Wrapf(err, "failed to NewBIP340PubKeyFromHex by %v", fpBtpPk)
	}

	// 1. get info from chain

	// 2. check if can as a valid fp

	// 3. store into storage

	// 4. update fp status.

	return nil
}
