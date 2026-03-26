package ext

func (s *Service) PausableName() string { return "messaging" }

func (s *Service) Pause() error {
	if s.messenger == nil {
		return nil
	}
	s.messenger.SetPaused(true)
	s.MarkPaused()
	return nil
}

func (s *Service) Resume() error {
	if s.messenger == nil {
		return nil
	}
	s.messenger.SetPaused(false)
	s.MarkResumed()
	return nil
}
