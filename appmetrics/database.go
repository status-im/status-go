package appmetrics

import (
	"database/sql"
	"errors"

	"github.com/xeipuuv/gojsonschema"
)

type AppMetricEventType string

type AppMetric struct {
	Event      AppMetricEventType `json:"event"`
	Value      string             `json:"value"`
	AppVersion string             `json:"app_version"`
	OS         string             `json:"os"`
}

const (
	// Events for testing the system
	TestEvent1 AppMetricEventType = "go/test1"
	TestEvent2 AppMetricEventType = "go/test2"

	// status-react navigation events
	NavigationNavigateToCofx AppMetricEventType = "navigation/navigate-to"
)

// EventSchemaMap Every event should have a schema attached
var EventSchemaMap = map[AppMetricEventType]interface{}{
	TestEvent1:               StringSchema,
	TestEvent2:               StringSchema,
	NavigationNavigateToCofx: NavigationNavigateToCofxSchema,
}

func NewDB(db *sql.DB) *Database {
	return &Database{db: db}
}

// Database sql wrapper for operations with browser objects.
type Database struct {
	db *sql.DB
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
}

func ValidateAppMetrics(appMetrics []AppMetric) (err error) {
	for _, metric := range appMetrics {
		schema := EventSchemaMap[metric.Event]

		if schema == nil {
			return errors.New("No schema defined for: " + string(metric.Event))
		}

		schemaLoader := gojsonschema.NewGoLoader(schema)
		valLoader := gojsonschema.NewStringLoader(metric.Value)
		res, err := gojsonschema.Validate(schemaLoader, valLoader)

		if err != nil {
			return err
		}

		if !res.Valid() {
			var errorDesc string = "Error in event: " + string(metric.Event) + "\n"
			for _, e := range res.Errors() {
				errorDesc = errorDesc + "value." + e.Context().String() + ":" + e.Description() + ", "
			}
			return errors.New(errorDesc)
		}
	}
	return
}

func (db *Database) SaveAppMetrics(appMetrics []AppMetric) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
	)

	// make sure that the shape of the metric is same as expected
	err = ValidateAppMetrics(appMetrics)
	if err != nil {
		return err
	}

	// start txn
	tx, err = db.db.Begin()
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

	insert, err = tx.Prepare("INSERT INTO app_metrics (event, value, app_version, operating_system) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}

	for _, metric := range appMetrics {
		_, err := insert.Exec(metric.Event, metric.Value, metric.AppVersion, metric.OS)

		if err != nil {
			return err
		}
	}
	return
}

func (db *Database) GetAppMetrics(limit int, offset int) (appMetrics []AppMetric, err error) {
	rows, err := db.db.Query("SELECT event, value, app_version, operating_system FROM app_metrics LIMIT ? OFFSET ?", limit, offset)

	if err != nil {
		return appMetrics, err
	}
	defer rows.Close()
	for rows.Next() {
		metric := AppMetric{}
		err := rows.Scan(&metric.Event, &metric.Value, &metric.AppVersion, &metric.OS)

		if err != nil {
			return appMetrics, err
		}
		appMetrics = append(appMetrics, metric)
	}
	return appMetrics, nil
}
