package connector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/rpc/network"
	"github.com/status-im/status-go/services/connector/chainutils"
)

const serviceName = "connector"

type Config struct {
	WSHost    string
	WSPort    int
	ProjectID string
}

func NewService(
	logger *zap.Logger,
	db *sql.DB,
	ethClientGetter chainutils.EthClientGetter,
	feeManager chainutils.FeeManager,
	nm *network.Manager,
	config *Config) *Service {
	s := &Service{
		logger:          logger,
		db:              db,
		ethClientGetter: ethClientGetter,
		feeManager:      feeManager,
		nm:              nm,
		config:          config,
	}
	s.api = NewAPI(s)
	return s
}

type Service struct {
	logger          *zap.Logger
	db              *sql.DB
	ethClientGetter chainutils.EthClientGetter
	feeManager      chainutils.FeeManager
	nm              *network.Manager

	// api stores a single API, to have the single *commands.ClientSideHandler instance.
	// This is more of a workaround and should be refactored together with the services refactoring.
	api *API

	config *Config

	rpcServer *gethrpc.Server
	wsServer  *http.Server
}

func (s *Service) Start() error {
	// Create an RPC server
	s.rpcServer = gethrpc.NewServer()

	for _, api := range s.APIs() {
		err := s.rpcServer.RegisterName(api.Namespace, api.Service)
		if err != nil {
			return err
		}
	}

	// Expose the RPC server over websocket
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead && r.URL.Path == "/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Inject connection type into request context
		ctx := WithConnectionType(r.Context(), ConnectionTypeUntrusted)
		r = r.WithContext(ctx)

		// FIXME: this is a temporary solution to allow all origins
		origins := []string{"*"}
		wsHandler := s.rpcServer.WebsocketHandler(origins)
		wsHandler.ServeHTTP(w, r)
	})
	s.wsServer = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.config.WSHost, s.config.WSPort),
		Handler:           handler,
		ReadHeaderTimeout: time.Second * 10,
	}

	go func() {
		defer common.LogOnPanic()
		err := s.wsServer.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("connector server closed with error", zap.Error(err))
		}
	}()

	return nil
}

func (s *Service) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if s.api != nil && s.api.wcClient != nil {
		if err := s.api.wcClient.Close(); err != nil {
			s.logger.Error("failed to close WalletConnect client", zap.Error(err))
		}
	}

	if s.wsServer == nil {
		return nil
	}

	err := s.wsServer.Shutdown(ctx)
	if err == nil {
		return nil
	}

	s.logger.Error("failed to stop status connector server", zap.Error(err))
	return s.wsServer.Close()
}

func (s *Service) APIs() []gethrpc.API {
	return []gethrpc.API{
		{
			Namespace: serviceName,
			Version:   "0.1.0",
			Service:   s.api,
		},
	}
}
