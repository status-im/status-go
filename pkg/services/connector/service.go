package connector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/rpc/network"
	"github.com/status-im/status-go/internal/signal"
	"github.com/status-im/status-go/pkg/services/connector/chainutils"
	persistence "github.com/status-im/status-go/pkg/services/connector/database"
	"github.com/status-im/status-go/pkg/services/connector/walletconnect"
)

const serviceName = "connector"

type Config struct {
	WSEnabled bool
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
	s.initWCClient()
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

	rpcServer          *gethrpc.Server
	wsServer           *http.Server
	mu                 sync.Mutex
	paused             bool
	started            bool
	purgeEphemeralOnce sync.Once

	wcClient atomic.Pointer[walletconnect.Client] // owned by Service; recreated after Pause/Resume or Stop
}

func (s *Service) GetClient() *walletconnect.Client {
	return s.wcClient.Load()
}

func (s *Service) restoreActiveWCSessions(wcClient *walletconnect.Client) {
	activeSessions, err := persistence.SelectActiveWCSessions(s.db, time.Now().Unix())
	if err != nil {
		s.logger.Error("failed to load active WC sessions", zap.Error(err))
		return
	}
	if len(activeSessions) == 0 {
		return
	}
	restored := make([]walletconnect.RestoredSession, 0, len(activeSessions))
	for _, session := range activeSessions {
		restored = append(restored, walletconnect.RestoredSession{
			Topic:  session.Topic,
			SymKey: session.SymKey,
		})
	}
	wcClient.RestoreSessions(restored)
	s.logger.Info("restored WalletConnect sessions", zap.Int("count", len(restored)))
	if err := wcClient.ConnectAndResubscribe(); err != nil {
		s.logger.Warn("failed to connect relay for restored WC sessions", zap.Error(err))
	}
}

// initWCClient creates wcClient when nil. Safe to call without holding s.mu (uses atomic.Pointer).
func (s *Service) initWCClient() {
	if s.wcClient.Load() != nil {
		return
	}

	wcClient, err := walletconnect.NewClient(s.config.ProjectID)
	if err != nil {
		s.logger.Error("failed to create WalletConnect client", zap.Error(err))
		return
	}
	if wcClient == nil {
		return
	}

	s.restoreActiveWCSessions(wcClient)

	wcClient.SetSessionDeleteHandler(func(topic string) {
		s.logger.Info("received wc_sessionDelete", zap.String("topic", topic))
		session, err := persistence.SelectWCSession(s.db, topic)
		dappURL := ""
		if err == nil && session != nil {
			dappURL = session.DAppURL
		}
		if err := s.api.wcSessionDisconnector.DisconnectSession(context.Background(), topic); err != nil {
			s.logger.Error("failed to disconnect WC session", zap.String("topic", topic), zap.Error(err))
		} else {
			s.logger.Info("WC session disconnected", zap.String("topic", topic), zap.String("dappURL", dappURL))
		}
		signal.SendWCSessionDelete(topic, dappURL)
	})

	if !s.wcClient.CompareAndSwap(nil, wcClient) {
		_ = wcClient.Close()
	}
}

func (s *Service) Start() error {
	s.initWCClient()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}

	s.purgeEphemeralOnce.Do(func() {
		if err := persistence.DeleteEphemeralDApps(s.db); err != nil {
			s.logger.Warn("failed to delete ephemeral dapps on first start", zap.Error(err))
		}
	})

	// Create an RPC server
	s.rpcServer = gethrpc.NewServer()

	for _, api := range s.APIs() {
		err := s.rpcServer.RegisterName(api.Namespace, api.Service)
		if err != nil {
			return err
		}
	}

	if !s.config.WSEnabled {
		s.started = true
		return nil
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
	wsServer := s.wsServer

	go func() {
		defer panics.LogOnPanic()
		err := wsServer.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("connector server closed with error", zap.Error(err))
		}
	}()
	s.started = true
	s.paused = false
	return nil
}

func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.stopLocked()
	if err2 := persistence.DeleteEphemeralDApps(s.db); err2 != nil {
		s.logger.Warn("failed to delete ephemeral dapps on stop", zap.Error(err2))
	}
	return err
}

func (s *Service) stopLocked() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if c := s.wcClient.Swap(nil); c != nil {
		if err := c.Close(); err != nil {
			s.logger.Error("failed to close WalletConnect client", zap.Error(err))
		}
	}

	if s.wsServer == nil {
		s.started = false
		return nil
	}

	err := s.wsServer.Shutdown(ctx)
	if err == nil {
		s.wsServer = nil
		s.started = false
		return nil
	}

	s.logger.Error("failed to stop status connector server", zap.Error(err))
	wsServer := s.wsServer
	s.wsServer = nil
	s.started = false
	return wsServer.Close()
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
