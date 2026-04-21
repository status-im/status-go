package timesource

func (s *ntpTimeSource) PausableName() string { return "ntp" }

func (s *ntpTimeSource) Pause() error {
	s.MarkPaused()
	return nil
}

func (s *ntpTimeSource) Resume() error {
	s.MarkResumed()
	return nil
}
