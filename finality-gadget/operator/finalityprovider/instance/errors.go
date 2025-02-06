package fp_instance

import (
	"fmt"

	bbntypes "github.com/babylonlabs-io/babylon/types"
)

type CriticalError struct {
	err     error
	fpBtcPk *bbntypes.BIP340PubKey
}

func (ce *CriticalError) Error() string {
	return fmt.Sprintf("critical err on finality-provider %s: %s", ce.fpBtcPk.MarshalHex(), ce.err.Error())
}
