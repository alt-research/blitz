package fp

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/errors"
	"github.com/lightningnetwork/lnd/kvdb"
	"go.uber.org/zap"

	bbntypes "github.com/babylonlabs-io/babylon/types"
	"github.com/babylonlabs-io/finality-provider/clientcontroller"
	"github.com/babylonlabs-io/finality-provider/eotsmanager"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/babylonlabs-io/finality-provider/finality-provider/service"
	"github.com/babylonlabs-io/finality-provider/finality-provider/store"
	"github.com/babylonlabs-io/finality-provider/metrics"
)

type FinalityProviderManager struct {
	config *fpcfg.Config

	fpIns *service.FinalityProviderInstance

	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup

	cc              clientcontroller.ClientController
	em              eotsmanager.EOTSManager
	fps             *store.FinalityProviderStore
	pubRandStore    *store.PubRandProofStore
	criticalErrChan chan *service.CriticalError

	quit chan struct{}

	metrics *metrics.FpMetrics
	logger  *zap.Logger
}

func NewFinalityProviderManager(
	ctx context.Context,
	config *fpcfg.Config,
	logger *zap.Logger,
	cc clientcontroller.ClientController,
	em eotsmanager.EOTSManager,
	db kvdb.Backend,
) (*FinalityProviderManager, error) {
	fpStore, err := store.NewFinalityProviderStore(db)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiate finality provider store: %w", err)
	}
	pubRandStore, err := store.NewPubRandProofStore(db)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiate public randomness store: %w", err)
	}

	fpMetrics := metrics.NewFpMetrics()

	return &FinalityProviderManager{
		config: config,

		cc:              cc,
		em:              em,
		fps:             fpStore,
		pubRandStore:    pubRandStore,
		criticalErrChan: make(chan *service.CriticalError),
		quit:            make(chan struct{}),

		metrics: fpMetrics,
		logger:  logger,
	}, nil

}

// startFinalityProviderInstance creates a finality-provider instance, starts it and adds it into the finality-provider manager
func (fpm *FinalityProviderManager) startFinalityProviderInstance(
	pk *bbntypes.BIP340PubKey,
	passphrase string,
) error {
	pkHex := pk.MarshalHex()
	if fpm.fpIns == nil {
		fpIns, err := service.NewFinalityProviderInstance(
			pk, fpm.config, fpm.fps, fpm.pubRandStore, fpm.cc, fpm.em,
			fpm.metrics, passphrase, fpm.criticalErrChan, fpm.logger,
		)
		if err != nil {
			return fmt.Errorf("failed to create finality provider instance %s: %w", pkHex, err)
		}

		fpm.fpIns = fpIns
	}

	return fpm.fpIns.Start()
}

// monitorCriticalErr takes actions when it receives critical errors from a finality-provider instance
// if the finality-provider is slashed, it will be terminated and the program keeps running in case
// new finality providers join
// otherwise, the program will panic
func (fpm *FinalityProviderManager) monitorCriticalErr(ctx context.Context) {
	defer fpm.wg.Done()

	var criticalErr *service.CriticalError

	for {
		select {
		case criticalErr = <-fpm.criticalErrChan:
			// TODO: handle critical
			fpm.logger.Sugar().Error("criticalErrChan", "err", criticalErr)
		case <-fpm.quit:
			return
		}
	}
}

func (fpm *FinalityProviderManager) Stop() error {
	var stopErr error
	fpm.stopOnce.Do(func() {
		close(fpm.quit)
		fpm.wg.Wait()

		if fpm.fpIns == nil {
			return
		}

		if !fpm.fpIns.IsRunning() {
			return
		}

		pkHex := fpm.fpIns.GetBtcPkHex()
		fpm.logger.Info("stopping finality provider", zap.String("pk", pkHex))

		if err := fpm.fpIns.Stop(); err != nil {
			stopErr = err
			return
		}

		fpm.logger.Info("finality provider is stopped", zap.String("pk", pkHex))
	})

	return stopErr
}

func (fpm *FinalityProviderManager) Start(
	ctx context.Context,
	pk *bbntypes.BIP340PubKey,
	passphrase string,
) error {
	if err := fpm.startFinalityProviderInstance(pk, passphrase); err != nil {
		return errors.Wrap(err, "startFinalityProviderInstance failed")
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		}
	}
}
