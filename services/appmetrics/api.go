package appmetrics

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/appmetrics"
)

func NewAppMetricsAPI (db *appmetrics.Database) *API {
	return &API{db}
}

type API struct {
	db *appmetrics.Database
}

func (api *API) ValidateAppMetrics(ctx context.Context, appMetrics []appmetrics.AppMetric) error {
	log.Info("[AppMetricsAPI::ValidateAppMetrics]")
	return api.db.ValidateAppMetrics(appMetrics)
}
