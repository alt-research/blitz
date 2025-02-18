package fp_instance

import (
	"context"
	"fmt"

	bbntypes "github.com/babylonlabs-io/babylon/types"
	"github.com/babylonlabs-io/finality-provider/finality-provider/service"
	"go.uber.org/zap"
)

type CriticalError struct {
	inner   *service.CriticalError
	err     error
	fpBtcPk *bbntypes.BIP340PubKey
}

func newCriticalError(err *service.CriticalError) *CriticalError {
	return &CriticalError{
		inner: err,
	}
}

func (ce *CriticalError) Error() string {
	if ce.err != nil {
		return ce.err.Error()
	}

	return fmt.Sprintf("critical err on finality-provider %s: %s", ce.fpBtcPk.MarshalHex(), ce.err.Error())
}

func NewCriticalErrorChan(
	ctx context.Context,
	logger *zap.Logger,
	errChan chan<- *CriticalError) chan<- *service.CriticalError {
	res := make(chan *service.CriticalError)
	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("exit by finishing for CriticalErrorChan Mapping ")
				return
			case err := <-res:
				logger.Sugar().Info("a error from inner chan by critical error mapping", "err", err)
				errChan <- newCriticalError(err)
			}
		}
	}()

	return res
}
