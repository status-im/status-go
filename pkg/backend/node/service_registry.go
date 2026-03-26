package node

import (
	"errors"
	"fmt"
	"sync"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/server"
)

// PausableMediaServer wraps a server.MediaServer to implement common.Pausable.
type PausableMediaServer struct {
	common.PauseBroadcaster
	s *server.MediaServer
}

func newPausableMediaServer(s *server.MediaServer) *PausableMediaServer {
	p := &PausableMediaServer{s: s}
	p.MarkStarted()
	return p
}

func (p *PausableMediaServer) PausableName() string { return "mediaserver" }

func (p *PausableMediaServer) Pause() error {
	p.s.ToBackground()
	p.MarkPaused()
	return nil
}

func (p *PausableMediaServer) Resume() error {
	p.s.ToForeground()
	p.MarkResumed()
	return nil
}

// PausableServiceInfo is a lightweight snapshot of a pausable service's state.
type PausableServiceInfo struct {
	Name  string              `json:"name"`
	State common.ServiceState `json:"state"`
}

// ServiceRegistry tracks services that support granular pause/resume control.
type ServiceRegistry struct {
	mu        sync.RWMutex
	pausables map[string]common.Pausable
}

func newServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		pausables: make(map[string]common.Pausable),
	}
}

// Register adds a Pausable service to the registry.
func (r *ServiceRegistry) Register(p common.Pausable) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pausables[p.PausableName()] = p
}

// ListPausable returns a snapshot of all registered services and their current state.
func (r *ServiceRegistry) ListPausable() []PausableServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]PausableServiceInfo, 0, len(r.pausables))
	for name, p := range r.pausables {
		result = append(result, PausableServiceInfo{Name: name, State: p.PausableState()})
	}
	return result
}

// Pause pauses the named service. Returns an error if the service is not found.
func (r *ServiceRegistry) Pause(name string) error {
	r.mu.RLock()
	p, ok := r.pausables[name]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("service %q not found in registry", name)
	}
	return p.Pause()
}

// Resume resumes the named service. Returns an error if the service is not found.
func (r *ServiceRegistry) Resume(name string) error {
	r.mu.RLock()
	p, ok := r.pausables[name]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("service %q not found in registry", name)
	}
	return p.Resume()
}

// PauseMultiple pauses all named services, collecting any errors.
func (r *ServiceRegistry) PauseMultiple(names []string) error {
	var errs []error
	for _, name := range names {
		if err := r.Pause(name); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// ResumeMultiple resumes all named services, collecting any errors.
func (r *ServiceRegistry) ResumeMultiple(names []string) error {
	var errs []error
	for _, name := range names {
		if err := r.Resume(name); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// PauseAll pauses every registered service.
func (r *ServiceRegistry) PauseAll() error {
	r.mu.RLock()
	names := make([]string, 0, len(r.pausables))
	for name := range r.pausables {
		names = append(names, name)
	}
	r.mu.RUnlock()
	return r.PauseMultiple(names)
}

// ResumeAll resumes every registered service.
func (r *ServiceRegistry) ResumeAll() error {
	r.mu.RLock()
	names := make([]string, 0, len(r.pausables))
	for name := range r.pausables {
		names = append(names, name)
	}
	r.mu.RUnlock()
	return r.ResumeMultiple(names)
}
