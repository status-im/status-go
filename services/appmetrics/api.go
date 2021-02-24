package appmetrics

import "context"

func NewAPI(db *Database) *API {
	return &API{db}
}

type API struct {
	db *Database
}

func (a *API) SaveAppMetrics(ctx context.Context, appMetrics []AppMetric) error {
	return a.db.SaveAppMetrics(appMetrics)
}
