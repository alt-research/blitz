package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/rs/cors"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/babylonlabs-io/finality-gadget/config"
	"github.com/babylonlabs-io/finality-gadget/db"

	"github.com/alt-research/blitz/finality-gadget/core/logging"
	finalitygadget "github.com/alt-research/blitz/finality-gadget/sdk/client"
)

// Server is the main daemon construct for the finality gadget server. It handles
// spinning up the RPC sever, the database, and any other components that the
// the finality gadget server needs to run.
type Server struct {
	rpcServer *rpcServer
	cfg       *config.Config
	db        db.IDatabaseHandler

	logger logging.Logger
	wg     sync.WaitGroup
}

// NewFinalityGadgetServer creates a new server with the given config.
func NewFinalityGadgetServer(cfg *config.Config, db db.IDatabaseHandler, fg finalitygadget.IFinalityGadget, logger logging.Logger) *Server {
	return &Server{
		cfg:       cfg,
		rpcServer: newRPCServer(fg),
		db:        db,
		logger:    logger,
	}
}

func (s *Server) Wait() {
	s.wg.Wait()
}

func (s *Server) Start(ctx context.Context) error {
	s.wg.Add(1)
	defer func() {
		s.logger.Info("Stop finality gadget operator rpc server service")
		s.logger.Info("Closing database...")
		s.db.Close()
		s.logger.Info("Database closed")
		s.wg.Done()
	}()

	s.logger.Info("Starting finality gadget operator rpc server service")

	// we create listeners from the GRPCListener defined in the config.
	lis, err := net.Listen("tcp", s.cfg.GRPCListener)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.cfg.GRPCListener, err)
	}
	defer lis.Close()

	// Create grpc server
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()
	if err := s.rpcServer.RegisterWithGrpcServer(grpcServer); err != nil {
		return fmt.Errorf("failed to register gRPC server: %w", err)
	}

	// All the necessary components have been registered, so we can
	// actually start listening for requests.
	if err := s.startGrpcListen(grpcServer, []net.Listener{lis}); err != nil {
		return fmt.Errorf("failed to start gRPC listener: %v", err)
	}

	// Add cors handler to allow local development
	corsHandler := cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool {
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			if u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" {
				return true
			}
			return false
		},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
	}).Handler(s.newHttpHandler())

	// Create http server.
	httpServer := &http.Server{
		Addr:    s.cfg.HTTPListener,
		Handler: corsHandler,
	}

	s.wg.Add(1)
	defer func() {
		s.logger.Info("Closing http server...")
		err := httpServer.Close()
		if err != nil {
			s.logger.Error("http server closed failed", "err", err)
		}
		s.wg.Done()
		s.logger.Info("Http server closed")
	}()

	go func() {
		s.logger.Info("Starting standalone HTTP server on port 8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server failed", zap.Error(err))
		}

		s.logger.Info("Finality gadget is active")
	}()

	// Wait ctx done
	<-ctx.Done()

	return nil
}

// startGrpcListen starts the GRPC server on the passed listeners.
func (s *Server) startGrpcListen(grpcServer *grpc.Server, listeners []net.Listener) error {

	// Use a WaitGroup so we can be sure the instructions on how to input the
	// password is the last thing to be printed to the console.
	var wg sync.WaitGroup

	for _, lis := range listeners {
		wg.Add(1)
		go func(lis net.Listener) {
			s.logger.Info("RPC server listening", zap.String("address", lis.Addr().String()))

			// Close the ready chan to indicate we are listening.
			defer lis.Close()

			wg.Done()
			_ = grpcServer.Serve(lis)
		}(lis)
	}

	// Wait for gRPC servers to be up running.
	wg.Wait()

	return nil
}

func (s *Server) newHttpHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transaction", s.txStatusHandler)
	mux.HandleFunc("/health", s.healthHandler)
	return mux
}
