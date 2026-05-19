package connector

import statuscommon "github.com/status-im/status-go/common"

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

func (s *Service) PausableState() statuscommon.ServiceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started && !s.paused {
		return statuscommon.ServiceStateStopped
	}
	if s.paused {
		return statuscommon.ServiceStatePaused
	}
	return statuscommon.ServiceStateRunning
}
