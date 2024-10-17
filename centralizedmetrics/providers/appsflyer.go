package providers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/centralizedmetrics/common"
)

const AppsflyerBaseURL = "https://api3.appsflyer.com"

var AppsflyerAppID = ""
var AppsflyerToken = ""

// AppsflyerMetricProcessor implements MetricProcessor for Appsflyer
type AppsflyerMetricProcessor struct {
	appID   string
	secret  string
	baseURL string

	logger *zap.Logger
}

// NewAppsflyerMetricProcessor is a constructor for AppsflyerMetricProcessor
func NewAppsflyerMetricProcessor(appID, secret, baseURL string, logger *zap.Logger) *AppsflyerMetricProcessor {
	return &AppsflyerMetricProcessor{
		appID:   appID,
		secret:  secret,
		baseURL: baseURL,
		logger:  logger,
	}
}

func (p *AppsflyerMetricProcessor) GetAppID() string {
	if len(p.appID) != 0 {
		return p.appID
	}
	return AppsflyerAppID
}

func (p *AppsflyerMetricProcessor) GetToken() string {
	if len(p.secret) != 0 {
		return p.secret
	}

	return AppsflyerToken
}

// Process processes an array of metrics and sends them to the Appsflyer API
func (p *AppsflyerMetricProcessor) Process(metrics []common.Metric) error {
	for _, metric := range metrics {
		if err := p.sendToAppsflyer(metric); err != nil {
			return err
		}
	}
	return nil
}

// sendToAppsflyer sends a single metric to the Appsflyer API
func (p *AppsflyerMetricProcessor) sendToAppsflyer(metric common.Metric) error {
	url := fmt.Sprintf("%s/inappevent/%s", p.baseURL, p.GetAppID())

	payload, err := json.Marshal(toAppsflyerMetric(metric))
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("authentication", p.GetToken())
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		p.logger.Warn("failed to send metric", zap.Int("status-code", resp.StatusCode), zap.String("body", string(body)), zap.Error(err))
		return errors.New("failed to send metric to Appsflyer")
	}

	return nil
}

func toAppsflyerMetric(metric common.Metric) appsflyerMetric {
	timestampMillis := metric.Timestamp

	seconds := timestampMillis / 1000
	nanoseconds := (timestampMillis % 1000) * int64(time.Millisecond)

	t := time.Unix(seconds, nanoseconds).UTC()

	formattedTime := t.Format("2006-01-02 15:04:05.000")

	return appsflyerMetric{
		AppsflyerID: metric.UserID,
		EventName:   metric.EventName,
		EventValue:  metric.EventValue,
		EventTime:   formattedTime,
	}
}

type appsflyerMetric struct {
	AppsflyerID string      `json:"appsflyer_id"`
	EventName   string      `json:"eventName"`
	EventValue  interface{} `json:"eventValue"`
	EventTime   string      `json:"eventTime"`
}
