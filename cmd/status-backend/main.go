package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/status-im/status-go/cmd/status-backend/server"
	logutils "github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/panics"
	statusgo "github.com/status-im/status-go/mobile"
	"github.com/status-im/status-go/pkg/sentry"
	"github.com/status-im/status-go/pkg/version"
)

var (
	address      = flag.String("address", "127.0.0.1:0", "host:port to listen")
	pprofEnabled = flag.Bool("pprof", false, "enable pprof")
	logger       *zap.Logger
)

func init() {
	logSettings := logutils.LogSettings{
		Enabled: true,
		Level:   "INFO",
	}
	if err := logutils.OverrideRootLoggerWithConfig(logSettings); err != nil {
		panic(err)
	}

	logger = logutils.ZapLogger()
}

func main() {
	sentry.MustInit(
		sentry.WithDefaultEnvironmentDSN(),
		sentry.WithContext("status-backend", version.Version()),
	)
	defer sentry.Recover()

	flag.Parse()
	go func() {
		defer panics.LogOnPanic()
		handleInterrupts()
	}()

	srv := server.NewServer(
		logger.Named("server"),
		server.WithProfiling(*pprofEnabled),
	)
	srv.Setup()

	err := srv.Listen(*address)
	if err != nil {
		logger.Error("failed to start server", zap.Error(err))
		return
	}

	logger.Info("status-backend started",
		zap.String("address", srv.Address()),
		zap.String("version", version.Version()),
		zap.String("gitCommit", version.GitCommit()),
	)
	srv.RegisterMobileAPI()
	srv.Serve()
}

// handleInterrupts catches interrupt signal (SIGTERM/SIGINT) and
// gracefully logouts and stops the node.
func handleInterrupts() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(ch)

	receivedSignal := <-ch
	logger.Info("interrupt signal received", zap.Stringer("signal", receivedSignal))
	_ = statusgo.Logout()
	os.Exit(0)
}
