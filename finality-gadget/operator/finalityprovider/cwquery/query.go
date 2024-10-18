package cwquery

import (
	"github.com/alt-research/blitz/finality-gadget/core/logging"
	"github.com/alt-research/blitz/finality-gadget/operator/finalityprovider/cwclient"
)

type CosmosWasmContractQuery struct {
	logger logging.Logger

	btcPkHex string
	cwClient cwclient.ICosmosWasmContractClient
}

func NewCosmosWasmContractQuery(logger logging.Logger, btcPkHex string, cwClient cwclient.ICosmosWasmContractClient) *CosmosWasmContractQuery {
	return &CosmosWasmContractQuery{
		logger:   logger,
		btcPkHex: btcPkHex,
		cwClient: cwClient,
	}
}
