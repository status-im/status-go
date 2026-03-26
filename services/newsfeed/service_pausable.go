package newsfeed

func (s *Service) PausableName() string { return "newsfeed" }

func (s *Service) Pause() error {
	if s.newsFeedManager != nil {
		s.newsFeedManager.StopPolling()
	}
	s.MarkPaused()
	return nil
}

func (s *Service) Resume() error {
	return s.Start()
}
