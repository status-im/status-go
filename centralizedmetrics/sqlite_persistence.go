package centralizedmetrics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/status-im/status-go/centralizedmetrics/common"
)

type SQLiteMetricRepository struct {
	db *sql.DB
}

func NewSQLiteMetricRepository(db *sql.DB) *SQLiteMetricRepository {
	return &SQLiteMetricRepository{db: db}
}

func (r *SQLiteMetricRepository) Poll() ([]common.Metric, error) {
	tx, err := r.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return nil, err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	userID, err := r.UserID(tx)
	if err != nil {
		return nil, err
	}

	rows, err := tx.Query("SELECT id, event_name, event_value, platform, app_version, timestamp FROM centralizedmetrics_metrics limit 10")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []common.Metric
	for rows.Next() {
		var metric common.Metric
		var eventValue string

		if err := rows.Scan(&metric.ID, &metric.EventName, &eventValue, &metric.Platform, &metric.AppVersion, &metric.Timestamp); err != nil {
			return nil, err
		}

		// Deserialize eventValue
		if err := json.Unmarshal([]byte(eventValue), &metric.EventValue); err != nil {
			return nil, err
		}

		metric.UserID = userID

		metrics = append(metrics, metric)
	}

	return metrics, rows.Err()
}

func (r *SQLiteMetricRepository) Delete(metrics []common.Metric) error {
	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	stmt, err := tx.Prepare("DELETE FROM centralizedmetrics_metrics WHERE id = ?")
	if err != nil {
		return err
	}

	defer stmt.Close()

	for _, metric := range metrics {
		if _, err := stmt.Exec(metric.ID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLiteMetricRepository) Add(metric common.Metric) error {
	eventValue, err := json.Marshal(metric.EventValue)
	if err != nil {
		return err
	}

	_, err = r.db.Exec("INSERT INTO centralizedmetrics_metrics (id, event_name, event_value, platform, app_version, timestamp) VALUES (?, ?, ?, ?, ?, ?)",
		metric.ID, metric.EventName, string(eventValue), metric.Platform, metric.AppVersion, time.Now().UnixNano()/int64(time.Millisecond))
	return err
}

func (r *SQLiteMetricRepository) UserID(tx *sql.Tx) (string, error) {
	var err error
	if tx == nil {
		tx, err = r.db.BeginTx(context.Background(), &sql.TxOptions{})
		if err != nil {
			return "", err
		}
		defer func() {
			if err == nil {
				err = tx.Commit()
				return
			}
			// don't shadow original error
			_ = tx.Rollback()
		}()
	}

	var userID string

	// Check if a UUID already exists in the table
	err = tx.QueryRow("SELECT uuid FROM centralizedmetrics_uuid LIMIT 1").Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// clean up err
			err = nil
			// Generate a new UUID
			newUUID := uuid.New().String()

			// Insert the new UUID into the table
			_, err := tx.Exec("INSERT INTO centralizedmetrics_uuid (uuid) VALUES (?)", newUUID)
			if err != nil {
				return "", fmt.Errorf("failed to insert new UUID: %v", err)
			}

			return newUUID, nil
		}
		return "", fmt.Errorf("failed to query for existing UUID: %v", err)
	}

	return userID, nil
}

func (r *SQLiteMetricRepository) ToggleEnabled(enabled bool) error {
	tx, err := r.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	// make sure row is present
	userID, err := r.UserID(tx)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE centralizedmetrics_uuid SET enabled = ?, user_confirmed = 1 WHERE uuid = ?", enabled, userID)
	if err != nil {
		return err
	}

	// if we are enabling them, nothing else to do
	if enabled {
		return nil
	}

	// otherwise clean up metrics that might have been collected in the meantime
	_, err = tx.Exec("DELETE FROM centralizedmetrics_metrics")
	return err

}

func (r *SQLiteMetricRepository) Info() (*MetricsInfo, error) {
	info := MetricsInfo{}
	err := r.db.QueryRow("SELECT enabled,user_confirmed FROM centralizedmetrics_uuid LIMIT 1").Scan(&info.Enabled, &info.UserConfirmed)
	if err == sql.ErrNoRows {
		return &info, nil
	}

	if err != nil {
		return nil, err
	}
	return &info, nil
}
