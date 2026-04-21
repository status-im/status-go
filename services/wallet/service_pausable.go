package wallet

import statuscommon "github.com/status-im/status-go/common"

func (s *Service) PausableName() string { return "wallet" }

func (s *Service) Pause() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started || s.paused {
		return nil
	}
	s.stopBackgroundWorkers()
	s.paused = true
	return nil
}

func (s *Service) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started || !s.paused {
		return nil
	}
	if err := s.startBackgroundWorkers(); err != nil {
		return err
	}
	s.paused = false
	return nil
}

func (s *Service) PausableState() statuscommon.ServiceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return statuscommon.ServiceStateStopped
	}
	if s.paused {
		return statuscommon.ServiceStatePaused
	}
	return statuscommon.ServiceStateRunning
}
