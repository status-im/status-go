package newsfeed

func (s *Service) PausableName() string { return "newsfeed" }

func (s *Service) Pause() error {
	if s.newsFeedManager != nil {
		s.newsFeedManager.StopPolling()
	}
	s.started = false
	s.MarkPaused()
	return nil
}

func (s *Service) Resume() error {
	if err := s.Start(); err != nil {
		return err
	}
	s.MarkResumed()
	return nil
}
