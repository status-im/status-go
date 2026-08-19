package connector

import "github.com/status-im/status-go/internal/pausable"

func (s *Service) PausableName() string { return "connector" }

func (s *Service) Pause() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.paused {
		return nil
	}
	err := s.stopLocked()
	if err == nil {
		s.paused = true
	}
	return err
}

func (s *Service) Resume() error {
	s.mu.Lock()
	if !s.paused {
		s.mu.Unlock()
		return nil
	}
	s.paused = false
	s.started = false
	s.mu.Unlock()
	s.initWCClient()
	return s.Start()
}

func (s *Service) PausableState() pausable.ServiceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started && !s.paused {
		return pausable.ServiceStateStopped
	}
	if s.paused {
		return pausable.ServiceStatePaused
	}
	return pausable.ServiceStateRunning
}
