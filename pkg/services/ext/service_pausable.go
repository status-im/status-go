package ext

func (s *Service) PausableName() string { return "messaging" }

func (s *Service) Pause() error {
	if m := s.currentMessenger(); m != nil {
		m.SetPaused(true)
	}
	s.MarkPaused()
	return nil
}

func (s *Service) Resume() error {
	if m := s.currentMessenger(); m != nil {
		m.SetPaused(false)
	}
	s.MarkResumed()
	return nil
}
