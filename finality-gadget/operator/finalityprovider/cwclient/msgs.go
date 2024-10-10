package cwclient

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

// contractQueryMsgs the type for msg to query for contract,
// from ./contracts/nitro-finality-gadget/src/msg.rs
type contractQueryMsgs struct {
	Admin              *queryAdmin              `json:"admin,omitempty"`
	BlockVoters        *queryBlockVoters        `json:"block_voters,omitempty"`
	Config             *queryConfig             `json:"config,omitempty"`
	FirstPubRandCommit *queryFirstPubRandCommit `json:"first_pub_rand_commit,omitempty"`
	LastPubRandCommit  *queryLastPubRandCommit  `json:"last_pub_rand_commit,omitempty"`
	IsEnabled          *queryisEnabled          `json:"is_enabled,omitempty"`
}

func (c contractQueryMsgs) Marshal() ([]byte, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal json")
	}

	return data, nil
}

type queryAdmin struct{}

type contractAdminResponse struct {
	Admin *string `json:"admin"`
}

type queryBlockVoters struct {
	Height uint64 `json:"height"`
	Hash   string `json:"hash"`
}

type queryConfig struct{}

type contractConfigResponse struct {
	ConsumerId      string `json:"consumer_id"`
	ActivatedHeight uint64 `json:"activated_height"`
}

type PubRandCommit struct {
	/// `start_height` is the height of the first commitment
	StartHeight uint64 `json:"start_height"`
	/// `num_pub_rand` is the number of committed public randomness
	NumPubRand uint64 `json:"num_pub_rand"`
	/// `commitment` is the value of the commitment.
	/// Currently, it's the root of the Merkle tree constructed by the public randomness
	Commitment string `json:"commitment"`
}

type queryFirstPubRandCommit struct {
	BtcPkHex string `json:"btc_pk_hex"`
}

type queryLastPubRandCommit struct {
	BtcPkHex string `json:"btc_pk_hex"`
}

type queryisEnabled struct{}

func newQueryAdminMsg() ([]byte, error) {
	return contractQueryMsgs{
		Admin: &queryAdmin{},
	}.Marshal()
}

func newQueryBlockVotersMsg(height uint64, hash common.Hash) ([]byte, error) {
	return contractQueryMsgs{
		BlockVoters: &queryBlockVoters{
			Height: height,
			Hash:   hash.Hex(),
		},
	}.Marshal()
}

func newQueryConfigMsg() ([]byte, error) {
	return contractQueryMsgs{
		Config: &queryConfig{},
	}.Marshal()
}

func newQueryFirstPubRandCommitMsg(btcPkHex string) ([]byte, error) {
	return contractQueryMsgs{
		FirstPubRandCommit: &queryFirstPubRandCommit{BtcPkHex: btcPkHex},
	}.Marshal()
}

func newQueryLastPubRandCommitMsg(btcPkHex string) ([]byte, error) {
	return contractQueryMsgs{
		LastPubRandCommit: &queryLastPubRandCommit{BtcPkHex: btcPkHex},
	}.Marshal()
}

func newQueryIsEnabledMsg() ([]byte, error) {
	return contractQueryMsgs{
		IsEnabled: &queryisEnabled{},
	}.Marshal()
}
