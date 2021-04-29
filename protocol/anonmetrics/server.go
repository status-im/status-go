package anonmetrics

import (
	"database/sql"
	"go.uber.org/zap"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"

	"github.com/status-im/status-go/postgres"
	"github.com/status-im/status-go/protocol/protobuf"
)

type ServerConfig struct {
	Enabled     bool
	PostgresURI string
}

type Server struct {
	Config     *ServerConfig
	Logger     *zap.Logger
	PostgresDB *sql.DB
}

func NewServer(postgresURI string, migrationResource *bindata.AssetSource) (*Server, error) {
	db, err := postgres.NewMigratedDB(postgresURI, migrationResource)
	if err != nil {
		return nil, err
	}

	return &Server{
		PostgresDB: db,
	}, nil
}

func (s *Server) Stop() error {
	return s.PostgresDB.Close()
}

func (s *Server) StoreMetrics(appMetricsBatch protobuf.AnonymousMetricBatch) (err error) {
	appMetrics, err := adaptProtoBatchToModels(appMetricsBatch)
	if err != nil {
		return err
	}

	var (
		tx     *sql.Tx
		insert *sql.Stmt
	)

	// start txn
	tx, err = s.PostgresDB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	//noinspection ALL
	query := `INSERT INTO app_metrics (message_id, event, value, app_version, operating_system, session_id, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (message_id) DO NOTHING;`

	insert, err = tx.Prepare(query)
	if err != nil {
		return err
	}

	for _, metric := range appMetrics {
		_, err = insert.Exec(
			metric.MessageID,
			metric.Event,
			metric.Value,
			metric.AppVersion,
			metric.OS,
			metric.SessionID,
			metric.CreatedAt,
			)
		if err != nil {
			return
		}
	}
	return
}
