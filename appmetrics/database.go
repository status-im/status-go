package appmetrics

import (
	"database/sql"
	"encoding/json"
	"errors"

	"strings"

	"github.com/xeipuuv/gojsonschema"
)

type AppMetricEventType string

// Value is `json.RawMessage` so we can send any json shape, including strings
// Validation is handled using JSON schemas defined in validators.go, instead of Golang structs
type AppMetric struct {
	Event      AppMetricEventType `json:"event"`
	Value      json.RawMessage    `json:"value"`
	AppVersion string             `json:"app_version"`
	OS         string             `json:"os"`
}

type AppMetricValidationError struct {
	Metric AppMetric
	Errors []gojsonschema.ResultError
}

const (
	// status-react navigation events
	NavigationNavigateToCofx AppMetricEventType = "navigation/navigate-to"
)

// EventSchemaMap Every event should have a schema attached
var EventSchemaMap = map[AppMetricEventType]interface{}{
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

func jsonschemaErrorsToError(validationErrors []AppMetricValidationError) error {
	var fieldErrors []string

	for _, appMetricValidationError := range validationErrors {
		metric := appMetricValidationError.Metric
		errors := appMetricValidationError.Errors

		var errorDesc string = "Error in event: " + string(metric.Event) + " - "
		for _, e := range errors {
			errorDesc = errorDesc + "value." + e.Context().String() + ":" + e.Description()
		}
		fieldErrors = append(fieldErrors, errorDesc)
	}

	return errors.New(strings.Join(fieldErrors, "/ "))
}

func (db *Database) ValidateAppMetrics(appMetrics []AppMetric) (err error) {
	var calculatedErrors []AppMetricValidationError
	for _, metric := range appMetrics {
		schema := EventSchemaMap[metric.Event]

		if schema == nil {
			return errors.New("No schema defined for: " + string(metric.Event))
		}

		schemaLoader := gojsonschema.NewGoLoader(schema)
		valLoader := gojsonschema.NewStringLoader(string(metric.Value))
		res, err := gojsonschema.Validate(schemaLoader, valLoader)

		if err != nil {
			return err
		}

		// validate all metrics and save errors
		if !res.Valid() {
			calculatedErrors = append(calculatedErrors, AppMetricValidationError{metric, res.Errors()})
		}
	}

	if len(calculatedErrors) > 0 {
		return jsonschemaErrorsToError(calculatedErrors)
	}
	return
}

func (db *Database) SaveAppMetrics(appMetrics []AppMetric) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
	)

	// make sure that the shape of the metric is same as expected
	err = db.ValidateAppMetrics(appMetrics)
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
		_, err = insert.Exec(metric.Event, metric.Value, metric.AppVersion, metric.OS)
		if err != nil {
			return
		}
	}
	return
}

func (db *Database) GetAppMetrics(limit int, offset int) (appMetrics []AppMetric, err error) {
	rows, err := db.db.Query("SELECT event, value, app_version, operating_system FROM app_metrics LIMIT ? OFFSET ?", limit, offset)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		metric := AppMetric{}
		err := rows.Scan(&metric.Event, &metric.Value, &metric.AppVersion, &metric.OS)

		if err != nil {
			return nil, err
		}
		appMetrics = append(appMetrics, metric)
	}
	return appMetrics, nil
}
