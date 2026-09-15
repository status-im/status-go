package server

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	logutils "github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/panics"
	statusgo "github.com/status-im/status-go/mobile"
	"github.com/status-im/status-go/pkg/version"
)

// Run configures logging, starts the status-backend server on address and
// blocks serving it. It is what cmd/status-backend's main runs after parsing
// its flags, and it also backs the StatusBackendRunServer export of the
// generated libstatus bindings, so a non-Go host can start the server.
func Run(address string, options ...Option) error {
	logSettings := logutils.LogSettings{
		Enabled: true,
		Level:   "INFO",
	}
	if err := logutils.OverrideRootLoggerWithConfig(logSettings); err != nil {
		return err
	}
	logger := logutils.ZapLogger()

	go func() {
		defer panics.LogOnPanic()
		handleInterrupts(logger)
	}()

	srv := NewServer(logger.Named("server"), options...)
	srv.Setup()

	if err := srv.Listen(address); err != nil {
		return err
	}

	logger.Info("status-backend started",
		zap.String("address", srv.Address()),
		zap.String("version", version.Version()),
		zap.String("gitCommit", version.GitCommit()),
	)
	srv.RegisterMobileAPI()
	srv.Serve()
	return nil
}

// handleInterrupts catches interrupt signal (SIGTERM/SIGINT) and
// gracefully logouts and stops the node.
func handleInterrupts(logger *zap.Logger) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(ch)

	receivedSignal := <-ch
	logger.Info("interrupt signal received", zap.Stringer("signal", receivedSignal))
	_ = statusgo.Logout()
	os.Exit(0)
}
