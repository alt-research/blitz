package eotsmanager

import (
	fpeotsmanager "github.com/babylonlabs-io/finality-provider/eotsmanager"
	"github.com/babylonlabs-io/finality-provider/eotsmanager/client"
	"github.com/babylonlabs-io/finality-provider/eotsmanager/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/pkg/errors"
)

var _ fpeotsmanager.EOTSManager = &EOTSManagerClient{}

type EOTSManagerClient struct {
	inner fpeotsmanager.EOTSManager
	cfg   Config
}

type Config struct {
	// The remote address for eotsmanager
	RemoteAddr string `yaml:"remote_address"`
}

// Create eots manager client
func NewEOTSManagerClient(cfg Config) (*EOTSManagerClient, error) {
	cli, err := client.NewEOTSManagerGRpcClient(cfg.RemoteAddr)
	if err != nil {
		return nil, errors.Wrapf(err, "create eotsmanager client failed: %v", cfg.RemoteAddr)
	}

	return &EOTSManagerClient{
		inner: cli,
		cfg:   cfg,
	}, nil
}

// CreateKey generates a key pair at the given name and persists it in storage.
// The key pair is formatted by BIP-340 (Schnorr Signatures)
// It fails if there is an existing key Info with the same name or public key.
func (e *EOTSManagerClient) CreateKey(name, passphrase, hdPath string) ([]byte, error) {
	return e.inner.CreateKey(name, passphrase, hdPath)
}

// CreateRandomnessPairList generates a list of Schnorr randomness pairs from
// startHeight to startHeight+(num-1) where num means the number of public randomness
// It fails if the finality provider does not exist or a randomness pair has been created before
// or passPhrase is incorrect
// NOTE: the randomness is deterministically generated based on the EOTS key, chainID and
// block height
func (e *EOTSManagerClient) CreateRandomnessPairList(uid []byte, chainID []byte, startHeight uint64, num uint32, passphrase string) ([]*btcec.FieldVal, error) {
	return e.inner.CreateRandomnessPairList(uid, chainID, startHeight, num, passphrase)
}

// KeyRecord returns the finality provider record
// It fails if the finality provider does not exist or passPhrase is incorrect
func (e *EOTSManagerClient) KeyRecord(uid []byte, passphrase string) (*types.KeyRecord, error) {
	return e.inner.KeyRecord(uid, passphrase)
}

// SignEOTS signs an EOTS using the private key of the finality provider and the corresponding
// secret randomness of the given chain at the given height
// It fails if the finality provider does not exist or there's no randomness committed to the given height
// or passPhrase is incorrect
func (e *EOTSManagerClient) SignEOTS(uid []byte, chainID []byte, msg []byte, height uint64, passphrase string) (*btcec.ModNScalar, error) {
	return e.inner.SignEOTS(uid, chainID, msg, height, passphrase)
}

// SignSchnorrSig signs a Schnorr signature using the private key of the finality provider
// It fails if the finality provider does not exist or the message size is not 32 bytes
// or passPhrase is incorrect
func (e *EOTSManagerClient) SignSchnorrSig(uid []byte, msg []byte, passphrase string) (*schnorr.Signature, error) {
	return e.inner.SignSchnorrSig(uid, msg, passphrase)
}

func (e *EOTSManagerClient) Close() error {
	return e.inner.Close()
}
