package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/alt-research/blitz/finality-gadget/client/l2eth"
)

type JsonRpcServer struct {
	logger  *zap.Logger
	handler JsonRpcHandler
	vhosts  []string
	cors    []string
	wg      *sync.WaitGroup
}

func NewJsonRpcServer(logger *zap.Logger, ethClient *l2eth.L2EthClient, vhosts []string, cors []string) *JsonRpcServer {
	return &JsonRpcServer{
		logger: logger,
		vhosts: vhosts,
		handler: JsonRpcHandler{
			logger:    logger,
			ethClient: ethClient,
		},
		cors: cors,
		wg:   &sync.WaitGroup{},
	}
}

func (s *JsonRpcServer) GetAPI() gethrpc.API {
	return gethrpc.API{
		Namespace: "eth",
		Service:   &s.handler,
	}
}

type loggerHandler struct {
	id     atomic.Uint64
	logger *zap.Logger
	next   http.Handler
}

func (h *loggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rpcId := h.id.Add(1)
	h.logger.Sugar().Debug("handle http request", "id", rpcId)
	h.next.ServeHTTP(w, r)
	h.logger.Sugar().Debug("handle http returned", "id", rpcId)
}

func (s *JsonRpcServer) StartServer(ctx context.Context, serverIpPortAddr string) {
	s.logger.Sugar().Info("Start JSON RPC Server", "addr", serverIpPortAddr)

	rpcAPI := []gethrpc.API{s.GetAPI()}

	srv := gethrpc.NewServer()
	srv.SetBatchLimits(node.DefaultConfig.BatchRequestLimit, node.DefaultConfig.BatchResponseMaxSize)
	err := node.RegisterApis(rpcAPI, []string{"eth"}, srv)
	if err != nil {
		s.logger.Sugar().Fatalf("Could not register API: %w", err)
	}
	handler := node.NewHTTPHandlerStack(srv, s.cors, s.vhosts, nil)

	handlerWithLogger := &loggerHandler{
		logger: s.logger,
		next:   handler,
	}

	httpServer, addr, err := node.StartHTTPEndpoint(serverIpPortAddr, gethrpc.DefaultHTTPTimeouts, handlerWithLogger)
	if err != nil {
		s.logger.Sugar().Fatalf("Could not start RPC api: %v", err)
	}

	extapiURL := fmt.Sprintf("http://%v/", addr)
	s.logger.Sugar().Info("HTTP endpoint opened", "url", extapiURL)

	serverErr := make(chan error, 1)

	s.wg.Add(1)
	defer s.wg.Done()

	select {
	case <-ctx.Done():
		s.logger.Sugar().Info("Stop JSON RPC Server by Done")
		err := httpServer.Shutdown(context.Background())
		if err != nil {
			s.logger.Sugar().Errorf("Stop JSON RPC Server by error: %v", err.Error())
		}
	case err = <-serverErr:
	}

	if err != nil {
		s.logger.Sugar().Error("JSON RPC Server serve stopped by error", "err", err)
	} else {
		s.logger.Sugar().Info("JSON RPC Server serve stopped")
	}
}

func (s *JsonRpcServer) Wait() {
	s.wg.Wait()
}

type JsonRpcHandler struct {
	logger    *zap.Logger
	ethClient *l2eth.L2EthClient
}

type InitOperatorResponse struct {
	Ok     bool   `json:"ok"`
	Reason string `json:"reason"`
}

func (h *JsonRpcHandler) GetBlockByNumber(
	ctx context.Context,
	number rpc.BlockNumber, fullTx bool,
) (map[string]json.RawMessage, error) {
	var raw map[string]json.RawMessage
	err := h.ethClient.Client.Client().CallContext(ctx, &raw, "eth_getBlockByNumber", number.String(), fullTx)
	if err != nil {
		return nil, err
	}

	h.logger.Sugar().Debugf("get block by number %v", number)
	h.logger.Sugar().Debugf("get block resp %v", raw)

	return raw, nil

}
