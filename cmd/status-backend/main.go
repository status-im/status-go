package main

import (
	"flag"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/cmd/status-backend/server"
	"github.com/status-im/status-go/logutils"
	statusgo "github.com/status-im/status-go/mobile"
	"github.com/status-im/status-go/pkg/sentry"
	"github.com/status-im/status-go/pkg/version"
)

var (
	address = flag.String("address", "127.0.0.1:0", "host:port to listen")
	logger  = log.New("package", "status-go/cmd/status-backend")
)

func init() {
	logSettings := logutils.LogSettings{
		Enabled:   true,
		Level:     "INFO",
		Colorized: terminal.IsTerminal(int(os.Stdin.Fd())),
	}
	if err := logutils.OverrideRootLoggerWithConfig(logSettings); err != nil {
		stdlog.Fatalf("failed to initialize log: %v", err)
	}
}

func main() {
	sentry.MustInit(
		sentry.WithDefaultEnvironmentDSN(),
		sentry.WithContext("status-backend", version.Version()),
	)
	defer sentry.Recover()

	flag.Parse()
	go handleInterrupts()

	srv := server.NewServer()
	srv.Setup()

	err := srv.Listen(*address)
	if err != nil {
		logger.Error("failed to start server", "error", err)
		return
	}

	log.Info("status-backend started",
		"address", srv.Address(),
		"version", version.Version(),
		"gitCommit", version.GitCommit(),
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
	logger.Info("interrupt signal received", "signal", receivedSignal)
	_ = statusgo.Logout()
	os.Exit(0)
}
