package main

import (
	"flag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/cmd/statusd/server"
)

var (
	address = flag.String("address", "", "host:port to listen")
	logger  = log.New("package", "status-go/cmd/status-backend")
)

func main() {
	srv := server.NewServer()
	srv.Setup()

	err := srv.Listen(*address)
	if err != nil {
		logger.Error("failed to start server", "error", err)
		return
	}

	log.Info("server started", "address", srv.Address())
	srv.RegisterMobileAPI()
	srv.Serve()
}
