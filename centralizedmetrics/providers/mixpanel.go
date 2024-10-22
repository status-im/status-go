package providers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/status-im/status-go/centralizedmetrics/common"
)

const MixpanelBaseURL = "https://api.mixpanel.com"

var MixpanelToken = ""
var MixpanelAppID = ""

// MixpanelMetricProcessor implements MetricProcessor for Mixpanel
type MixpanelMetricProcessor struct {
	appID   string
	secret  string
	baseURL string

	logger *zap.Logger
}

// NewMixpanelMetricProcessor is a constructor for MixpanelMetricProcessor
func NewMixpanelMetricProcessor(appID, secret, baseURL string, logger *zap.Logger) *MixpanelMetricProcessor {
	return &MixpanelMetricProcessor{
		appID:   appID,
		secret:  secret,
		baseURL: baseURL,
		logger:  logger,
	}
}

func (amp *MixpanelMetricProcessor) GetAppID() string {
	if len(amp.appID) != 0 {
		return amp.appID
	}
	return MixpanelAppID
}

func (amp *MixpanelMetricProcessor) GetToken() string {
	if len(amp.secret) != 0 {
		return amp.secret
	}

	return MixpanelToken
}

// Process processes an array of metrics and sends them to the Mixpanel API
func (amp *MixpanelMetricProcessor) Process(metrics []common.Metric) error {
	if err := amp.sendToMixpanel(metrics); err != nil {
		return err
	}
	return nil
}

// sendToMixpanel sends a single metric to the Mixpanel API
func (amp *MixpanelMetricProcessor) sendToMixpanel(metrics []common.Metric) error {
	url := fmt.Sprintf("%s/track?project_id=%s&strict=1", amp.baseURL, amp.GetAppID())

	var mixPanelMetrics []mixpanelMetric

	for _, metric := range metrics {
		mixPanelMetrics = append(mixPanelMetrics, toMixpanelMetric(metric, amp.GetToken()))
	}
	payload, err := json.Marshal(mixPanelMetrics)
	if err != nil {
		return err
	}

	amp.logger.Info("sending metrics to", zap.String("url", url), zap.Any("metric", mixPanelMetrics), zap.String("secret", amp.GetToken()))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		amp.logger.Warn("failed to send metric", zap.Int("status-code", resp.StatusCode), zap.String("body", string(body)), zap.Error(err))
		return errors.New("failed to send metric to Mixpanel")
	}

	return nil
}

func toMixpanelMetric(metric common.Metric, token string) mixpanelMetric {

	properties := mixpanelMetricProperties{
		Time:                 metric.Timestamp,
		UserID:               metric.UserID,
		Platform:             metric.Platform,
		InsertID:             metric.ID,
		AppVersion:           metric.AppVersion,
		Token:                token,
		AdditionalProperties: metric.EventValue,
	}

	return mixpanelMetric{
		Event:      metric.EventName,
		Properties: properties,
	}
}

type mixpanelMetricProperties struct {
	Time                 int64          `json:"time"`
	UserID               string         `json:"distinct_id"`
	InsertID             string         `json:"$insert_id"`
	Platform             string         `json:"platform"`
	AppVersion           string         `json:"app_version"`
	AdditionalProperties map[string]any `json:"-"`
	Token                string         `json:"token"`
}

type mixpanelMetric struct {
	Event      string                   `json:"event"`
	Properties mixpanelMetricProperties `json:"properties"`
}

func (p mixpanelMetricProperties) MarshalJSON() ([]byte, error) {
	// Create a map and marshal the struct fields into it
	type alias mixpanelMetricProperties // Alias to avoid recursion
	data, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON into a map
	var mmpMap map[string]any
	if err := json.Unmarshal(data, &mmpMap); err != nil {
		return nil, err
	}

	// Merge AdditionalProperties into the map
	for key, value := range p.AdditionalProperties {
		mmpMap[key] = value
	}

	// Marshal the merged map back to JSON
	marshaled, err := json.Marshal(mmpMap)

	return marshaled, err
}

func (p *mixpanelMetricProperties) UnmarshalJSON(data []byte) error {
	// Create a temporary alias type to unmarshal known fields
	type alias mixpanelMetricProperties
	aux := &struct {
		*alias
	}{
		alias: (*alias)(p),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Unmarshal into a map to capture additional properties
	var rawMap map[string]any
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	// Remove known fields from the map
	delete(rawMap, "time")
	delete(rawMap, "token")
	delete(rawMap, "distinct_id")
	delete(rawMap, "$insert_id")
	delete(rawMap, "platform")
	delete(rawMap, "app_version")

	// Assign the remaining fields to AdditionalProperties
	p.AdditionalProperties = rawMap

	return nil
}
