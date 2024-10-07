package eotsmanager

import (
	fpeotsmanager "github.com/babylonlabs-io/finality-provider/eotsmanager"
	"github.com/babylonlabs-io/finality-provider/eotsmanager/client"
	"github.com/pkg/errors"
)

var _ fpeotsmanager.EOTSManager = &EOTSManagerClient{}

type EOTSManagerClient struct {
	fpeotsmanager.EOTSManager
}

func NewEOTSManagerClient(remoteAddr string) (*EOTSManagerClient, error) {
	cli, err := client.NewEOTSManagerGRpcClient(remoteAddr)
	if err != nil {
		return nil, errors.Wrapf(err, "create eotsmanager client failed: %v", remoteAddr)
	}

	return &EOTSManagerClient{
		EOTSManager: cli,
	}, nil
}
