package anonmetrics

import (
	"database/sql"
	"go.uber.org/zap"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"

	"github.com/status-im/status-go/appmetrics"
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
	if s.PostgresDB != nil {
		return s.PostgresDB.Close()
	}
	return nil
}

func (s *Server) StoreMetrics(appMetricsBatch protobuf.AnonymousMetricBatch) (appMetrics []*appmetrics.AppMetric, err error) {
	s.Logger.Debug("StoreMetrics() triggered with payload",
		zap.Reflect("appMetricsBatch", appMetricsBatch))
	appMetrics, err = adaptProtoBatchToModels(appMetricsBatch)
	if err != nil {
		return
	}

	var (
		tx     *sql.Tx
		insert *sql.Stmt
	)

	// start txn
	tx, err = s.PostgresDB.Begin()
	if err != nil {
		return
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
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (message_id) DO NOTHING;`

	insert, err = tx.Prepare(query)
	if err != nil {
		return
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

func (s *Server) getFromRows(rows *sql.Rows) (appMetrics []appmetrics.AppMetric, err error) {
	for rows.Next() {
		metric := appmetrics.AppMetric{}
		err = rows.Scan(
			&metric.ID,
			&metric.MessageID,
			&metric.Event,
			&metric.Value,
			&metric.AppVersion,
			&metric.OS,
			&metric.SessionID,
			&metric.CreatedAt,
			&metric.Processed,
			&metric.ReceivedAt,
		)
		if err != nil {
			return nil, err
		}
		appMetrics = append(appMetrics, metric)
	}
	return appMetrics, nil
}

func (s *Server) GetAppMetrics(limit int, offset int) ([]appmetrics.AppMetric, error) {
	rows, err := s.PostgresDB.Query("SELECT id, message_id, event, value, app_version, operating_system, session_id, created_at, processed, received_at FROM app_metrics LIMIT $1 OFFSET $2", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.getFromRows(rows)
}
