package eotsmanager

import (
	fpeotsmanager "github.com/babylonlabs-io/finality-provider/eotsmanager"
	"github.com/babylonlabs-io/finality-provider/eotsmanager/client"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var _ fpeotsmanager.EOTSManager = &EOTSManagerClient{}

type EOTSManagerClient struct {
	inner  fpeotsmanager.EOTSManager
	cfg    Config
	logger *zap.Logger
}

type Config struct {
	// The remote address for eotsmanager
	RemoteAddr string `yaml:"remote_address"`
}

func (c *Config) WithEnv() {
	// TODO
}

// Create eots manager client
func NewEOTSManagerClient(logger *zap.Logger, cfg Config) (*EOTSManagerClient, error) {
	cli, err := client.NewEOTSManagerGRpcClient(cfg.RemoteAddr)
	if err != nil {
		return nil, errors.Wrapf(err, "create eotsmanager client failed: %v", cfg.RemoteAddr)
	}

	return &EOTSManagerClient{
		inner:  cli,
		cfg:    cfg,
		logger: logger,
	}, nil
}

// CreateRandomnessPairList generates a list of Schnorr randomness pairs from
// startHeight to startHeight+(num-1) where num means the number of public randomness
// It fails if the finality provider does not exist or a randomness pair has been created before
// or passPhrase is incorrect
// NOTE: the randomness is deterministically generated based on the EOTS key, chainID and
// block height
func (e *EOTSManagerClient) CreateRandomnessPairList(uid []byte, chainID []byte, startHeight uint64, num uint32) ([]*btcec.FieldVal, error) {
	res, err := e.inner.CreateRandomnessPairList(uid, chainID, startHeight, num)
	e.logger.Sugar().Debugf("CreateRandomnessPairList %v %v", startHeight, res)

	return res, err
}

// SignEOTS signs an EOTS using the private key of the finality provider and the corresponding
// secret randomness of the given chain at the given height
// It fails if the finality provider does not exist or there's no randomness committed to the given height
// or passPhrase is incorrect
func (e *EOTSManagerClient) SignEOTS(uid []byte, chainID []byte, msg []byte, height uint64) (*btcec.ModNScalar, error) {
	return e.inner.SignEOTS(uid, chainID, msg, height)
}

// UnsafeSignEOTS should only be used in e2e tests for demonstration purposes.
// Does not offer double sign protection.
// Use SignEOTS for real operations.
func (e *EOTSManagerClient) UnsafeSignEOTS(uid []byte, chainID []byte, msg []byte, height uint64) (*btcec.ModNScalar, error) {
	return e.inner.UnsafeSignEOTS(uid, chainID, msg, height)
}

// SignSchnorrSig signs a Schnorr signature using the private key of the finality provider
// It fails if the finality provider does not exist or the message size is not 32 bytes
// or passPhrase is incorrect
func (e *EOTSManagerClient) SignSchnorrSig(uid []byte, msg []byte) (*schnorr.Signature, error) {
	return e.inner.SignSchnorrSig(uid, msg)
}

func (e *EOTSManagerClient) Close() error {
	return e.inner.Close()
}
