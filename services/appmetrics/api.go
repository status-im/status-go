package appmetrics

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/appmetrics"
)

func NewAPI (db *appmetrics.Database, metricsBufferedChan chan appmetrics.AppMetric) *API {
	return &API{db: db, metricsBufferedChan: metricsBufferedChan}
}

type API struct {
	db *appmetrics.Database
	metricsBufferedChan chan appmetrics.AppMetric
}

func (api *API) ValidateAppMetrics(ctx context.Context, appMetrics []appmetrics.AppMetric) error {
	log.Info("[AppMetricsAPI::ValidateAppMetrics]")
	return api.db.ValidateAppMetrics(appMetrics)
}

func (api *API) SaveAppMetrics(ctx context.Context, appMetrics []appmetrics.AppMetric) error {
	log.Info("[AppMetricsAPI::SaveAppMetrics]")
	chanFull := len(api.metricsBufferedChan) == cap(api.metricsBufferedChan)
	if chanFull {
		// channel is full, write all items to db, including the newly sent metrics
		for len(api.metricsBufferedChan) > 0 {
			appMetrics= append(appMetrics, <-api.metricsBufferedChan)
		}
		return api.db.SaveAppMetrics(appMetrics)
	} else {
		// there is space on the channel, write here, not in db
		for _, m := range(appMetrics) {
			api.metricsBufferedChan <- m
		}
	}
	return nil
}

func (api *API) GetAppMetrics(ctx context.Context, limit int, offset int) ([]appmetrics.AppMetric, error) {
	log.Info("[AppMetricsAPI::GetAppMetrics]")
	return api.db.GetAppMetrics(limit, offset)
}

