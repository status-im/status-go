package centralizedmetrics

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/centralizedmetrics/common"
	"github.com/status-im/status-go/t/helpers"
)

func openTestDB() (*sql.DB, error) {
	db, err := helpers.SetupTestMemorySQLAccountsDB(appdatabase.DbInitializer{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func setupTestDB(t *testing.T) *sql.DB {
	db, err := openTestDB()
	require.NoError(t, err)

	return db
}

func TestNewSQLiteMetricRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteMetricRepository(db)
	require.NotNil(t, repo)
	require.Equal(t, db, repo.db)
}

func TestSQLiteMetricRepository_Add(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteMetricRepository(db)

	metric := common.Metric{
		ID:        "id",
		UserID:    "user123",
		EventName: "purchase",
		EventValue: map[string]interface{}{
			"amount": 99.99,
		},
	}

	err := repo.Add(metric)
	require.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM centralizedmetrics_metrics WHERE id = ?", metric.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestSQLiteMetricRepository_Poll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteMetricRepository(db)

	// Insert test data
	metric := common.Metric{
		ID:        "id",
		UserID:    "user123",
		EventName: "purchase",
		EventValue: map[string]interface{}{
			"amount": 99.99,
		},
	}

	err := repo.Add(metric)
	require.NoError(t, err)

	metrics, err := repo.Poll()
	require.NoError(t, err)
	require.Len(t, metrics, 1)
	require.Equal(t, metric.ID, metrics[0].ID)
	require.Equal(t, metric.EventName, metrics[0].EventName)
	require.Equal(t, metric.EventValue, metrics[0].EventValue)
	require.NotEmpty(t, metrics[0].UserID)
	require.NotEmpty(t, metrics[0].Timestamp)
}

func TestSQLiteMetricRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteMetricRepository(db)

	// Insert test data
	metric := common.Metric{
		ID:        "id",
		EventName: "purchase",
		EventValue: map[string]interface{}{
			"amount": 99.99,
		},
	}

	err := repo.Add(metric)
	require.NoError(t, err)

	metrics := []common.Metric{metric}
	err = repo.Delete(metrics)
	require.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM centralizedmetrics_metrics WHERE id = ?", metric.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestSQLiteMetricRepository_UserID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteMetricRepository(db)

	// Test when there is no UUID in the table
	userID, err := repo.UserID(nil)
	require.NoError(t, err)
	require.NotEmpty(t, userID)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM centralizedmetrics_uuid WHERE uuid = ?", userID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Test when a UUID already exists
	existingUUID := userID
	userID, err = repo.UserID(nil)
	require.NoError(t, err)
	require.Equal(t, existingUUID, userID)
}

func TestSQLiteMetricRepository_Enabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteMetricRepository(db)

	info, err := repo.Info()
	require.NoError(t, err)
	require.False(t, info.Enabled)

	err = repo.ToggleEnabled(true)
	require.NoError(t, err)

	info, err = repo.Info()
	require.NoError(t, err)
	require.True(t, info.Enabled)

	err = repo.ToggleEnabled(false)
	require.NoError(t, err)

	info, err = repo.Info()
	require.NoError(t, err)
	require.False(t, info.Enabled)
}

func TestSQLiteMetricRepository_EnabledDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteMetricRepository(db)

	info, err := repo.Info()
	require.NoError(t, err)
	require.False(t, info.Enabled)

	metric := common.Metric{
		ID:        "id",
		UserID:    "user123",
		EventName: "purchase",
		EventValue: map[string]interface{}{
			"amount": 99.99,
		},
	}

	err = repo.Add(metric)
	require.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM centralizedmetrics_metrics WHERE id = ?", metric.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = repo.ToggleEnabled(false)
	require.NoError(t, err)

	err = db.QueryRow("SELECT COUNT(*) FROM centralizedmetrics_metrics WHERE id = ?", metric.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	info, err = repo.Info()
	require.NoError(t, err)
	require.False(t, info.Enabled)
	require.True(t, info.UserConfirmed)
}
