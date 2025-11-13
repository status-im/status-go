package backend

import (
	"encoding/json"
	"errors"

	"github.com/status-im/status-go/centralizedmetrics"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/requests"
)

var statusBackend *StatusBackend

func RPC(request string) string {
	return statusBackend.CallInProcessRPC(request)
}

type InitializeRequest struct {
	DataDir string `json:"rootDataDir"`
}

type InitializeResponse struct {
	Error                  error                           `json:"error"`
	Accounts               []multiaccounts.Account         `json:"accounts"`
	CentralizedMetricsInfo *centralizedmetrics.MetricsInfo `json:"centralizedMetricsInfo"`
}

// Initialize is the single point of StatusBackend instantiation with the provided configuration.
// An instance of StatusBackend is created as a global singleton, which shall only be used from C-bindings.
func Initialize(requestJSON string) string {
	response, err := initialize(requestJSON)
	if err != nil {
		response = &InitializeResponse{
			Error: err,
		}
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return `{"error": "failed to marshal response"}`
	}

	return string(responseJSON)
}

func initialize(requestJSON string) (*InitializeResponse, error) {
	if statusBackend != nil {
		return nil, errors.New("statusBackend already initialized")
	}

	var request requests.InitializeApplication
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return nil, errors.New("")
	}

	err = request.Validate()
	if err != nil {
		return nil, err
	}

	// Prepare StatusBackend options based on the request
	opts := []Option{
		WithLogger(
			logutils.ZapLogger().Named("backend")),
		WithMediaServer(
			*request.MediaServerAddress,
			request.MediaServerEnableTLS,
			request.MediaServerAdvertizeHost,
			request.MediaServerAdvertizePort),
		WithCentralizedMetrics(
			request.MixpanelAppID,
			request.MixpanelToken),
		WithSentry(
			request.SentryDSN),
		WithWakuFleets( // FIXME: Move to Messenger service
			request.WakuFleetsConfigFilePath,
			request.PushFleetsConfigFilePath),
	}
	if request.LogDir != "" {
		opts = append(opts, WithLogsDir(request.LogDir))
	}
	if request.LogLevel != "" {
		opts = append(opts, WithLogLevel(request.LogLevel))
	}
	if request.MetricsEnabled {
		opts = append(opts, WithMetrics(request.MetricsAddress))
	}
	if request.APILoggingEnabled {
		opts = append(opts, WithAPILogging())
	}

	// Create a global StatusBackend instance
	statusBackend, err = NewStatusBackend(request.DataDir, opts...)
	if err != nil {
		return nil, err
	}

	// Get metrics info for the response
	metricsInfo, err := statusBackend.CentralizedMetricsInfo()
	if err != nil {
		return nil, err
	}

	// Get list of existing users (accounts in multiaccounts db) for the response
	accs, err := statusBackend.RootService().ListAccounts()
	if err != nil {
		return nil, err
	}

	return &InitializeResponse{
		Accounts:               accs,
		CentralizedMetricsInfo: metricsInfo,
	}, nil
}
