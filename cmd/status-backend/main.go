package main

import (
	"flag"

	"go.uber.org/zap"

	"github.com/status-im/status-go/cmd/status-backend/server"
	logutils "github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/pkg/sentry"
	"github.com/status-im/status-go/pkg/version"
)

var (
	address      = flag.String("address", "127.0.0.1:0", "host:port to listen")
	pprofEnabled = flag.Bool("pprof", false, "enable pprof")
)

func main() {
	sentry.MustInit(
		sentry.WithDefaultEnvironmentDSN(),
		sentry.WithContext("status-backend", version.Version()),
	)
	defer sentry.Recover()

	flag.Parse()

	if err := server.Run(*address, server.WithProfiling(*pprofEnabled)); err != nil {
		logutils.ZapLogger().Error("failed to start server", zap.Error(err))
	}
}
