package linkpreview

import (
	"github.com/ethereum/go-ethereum/rpc"
	"go.uber.org/zap"

	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/services/linkpreview/unfurlers"
)

type Service struct {
	logger             *zap.Logger
	storage            Settings
	statusDataProvider unfurlers.StatusDataProvider
}

func NewService(logger *zap.Logger, settingsProvider Settings, statusDataProvider unfurlers.StatusDataProvider) *Service {
	return &Service{
		logger:             logger,
		storage:            settingsProvider,
		statusDataProvider: statusDataProvider,
	}
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "linkpreview",
			Version:   "1.0",
			Service: &PublicAPI{
				service: s,
			},
		},
	}
}

func (s *Service) SetStatusDataProvider(provider unfurlers.StatusDataProvider) {
	s.statusDataProvider = provider
}

func (s *Service) GetTextURLsToUnfurl(text string) *URLsUnfurlPlan {
	mode, err := s.storage.GetUnfurlingMode()
	if err != nil {
		// log the error and keep parsing the text
		s.logger.Error("GetTextURLsToUnfurl: failed to get settings", zap.Error(err))
		mode = settings.URLUnfurlingDisableAll
	}
	return GetTextURLsToUnfurl(text, mode)
}

func (s *Service) UnfurlURLs(urls []string) (UnfurlURLsResponse, error) {
	return UnfurlURLs(urls, nil, s.statusDataProvider, s.logger)
}
