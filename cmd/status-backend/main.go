package main

import (
	"flag"
	stdlog "log"
	"os"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/cmd/status-backend/server"
	"github.com/status-im/status-go/internal/sentry"
	"github.com/status-im/status-go/internal/version"
	"github.com/status-im/status-go/logutils"
)

var (
	address = flag.String("address", "127.0.0.1:0", "host:port to listen")
	logger  = log.New("package", "status-go/cmd/status-backend")
)

func init() {
	logSettings := logutils.LogSettings{
		Enabled:      true,
		MobileSystem: false,
		Level:        "INFO",
		Colorized:    terminal.IsTerminal(int(os.Stdin.Fd())),
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
