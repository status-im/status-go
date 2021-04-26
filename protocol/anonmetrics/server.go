package anonmetrics

import (
	"go.uber.org/zap"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"

	"github.com/status-im/status-go/postgres"
)

type ServerConfig struct {
	Enabled     bool
	PostgresURI string
}

type Server struct {
	Config     *ServerConfig
	Logger     *zap.Logger
	PostgresDB *postgres.DB
}

func NewServer(postgresURI string, migrationResource *bindata.AssetSource) (*Server, error) {
	db, err := postgres.NewPostgresDB(postgresURI, migrationResource)
	if err != nil {
		return nil, err
	}

	return &Server{
		PostgresDB: db,
	}, nil
}
