package requests

import (
	"errors"
)

var (
	ErrInitializeApplicationInvalidDataDir = errors.New("initialize-centralized-metric: no dataDir")
)

type InitializeApplication struct {
	DataDir       string `json:"dataDir"`
	MixpanelAppID string `json:"mixpanelAppId"`
	MixpanelToken string `json:"mixpanelToken"`
	// MediaServerEnableTLS is optional, if not provided, media server will use TLS by default
	MediaServerEnableTLS *bool  `json:"mediaServerEnableTLS"`
	SentryDSN            string `json:"sentryDSN"`

	// LogDir specifies the directory where logs are stored.
	// If empty, logs are stored in the `DataDir`.
	LogDir string `json:"logDir"`

	LogEnabled        bool   `json:"logEnabled"`
	LogLevel          string `json:"logLevel"`
	APILoggingEnabled bool   `json:"apiLoggingEnabled"`

	MetricsEnabled bool   `json:"metricsEnabled"`
	MetricsAddress string `json:"metricsAddress"`

	// WakuFleetsConfigFilePath specifies the file path for configuring fleets supported by the app.
	// File structure must be as params.FleetsMap.
	// When successfully loaded, overrides all hard-coded fleets with file contents.
	WakuFleetsConfigFilePath string `json:"wakuFleetsConfigFilePath"`
}

func (i *InitializeApplication) Validate() error {
	if len(i.DataDir) == 0 {
		return ErrInitializeApplicationInvalidDataDir
	}
	return nil
}
