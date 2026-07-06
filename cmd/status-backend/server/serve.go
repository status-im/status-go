package server

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	logutils "github.com/status-im/status-go/internal/logutils"
	statusgo "github.com/status-im/status-go/mobile"
	"github.com/status-im/status-go/pkg/version"
)

// Run starts the status-backend server on address and blocks serving it.
// It is the embeddable equivalent of cmd/status-backend's main (minus flag
// parsing, sentry and pprof) and backs the StatusBackendRunServer C export in
// the generated libstatus bindings, so the server can be started from a
// non-Go host such as the nimble-installed status_backend binary.
func Run(address string) error {
	logSettings := logutils.LogSettings{
		Enabled: true,
		Level:   "INFO",
	}
	if err := logutils.OverrideRootLoggerWithConfig(logSettings); err != nil {
		return err
	}
	logger := logutils.ZapLogger()

	go func() {
		defer common.LogOnPanic()
		handleInterrupts(logger)
	}()

	srv := NewServer(logger.Named("server"))
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
