package node

import (
	"errors"
	"fmt"
	"sync"

	"github.com/status-im/status-go/internal/pausable"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/services/media"
)

// ErrServiceNotFound is returned by Pause and Resume when the name is not registered.
var ErrServiceNotFound = errors.New("service not found in registry")

// PausableMediaServer wraps a media.Server to implement pausable.Pausable.
type PausableMediaServer struct {
	pausable.PauseBroadcaster
	s *media.Server
}

func newPausableMediaServer(s *media.Server) *PausableMediaServer {
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

// PausableMessenger wraps a protocol.Messenger to implement pausable.Pausable.
// Pause() → SetPaused(true) and Resume() → SetPaused(false)
type PausableMessenger struct {
	pausable.PauseBroadcaster
	m *protocol.Messenger
}

func newPausableMessenger(m *protocol.Messenger) *PausableMessenger {
	p := &PausableMessenger{m: m}
	p.MarkStarted()
	return p
}

func (p *PausableMessenger) PausableName() string { return "messenger" }

func (p *PausableMessenger) Pause() error {
	p.m.SetPaused(true)
	p.MarkPaused()
	return nil
}

func (p *PausableMessenger) Resume() error {
	p.m.SetPaused(false)
	p.MarkResumed()
	return nil
}

// PausableServiceInfo is a lightweight snapshot of a pausable service's state.
type PausableServiceInfo struct {
	Name  string                `json:"name"`
	State pausable.ServiceState `json:"state"`
}

// ServiceRegistry tracks services that support granular pause/resume control.
type ServiceRegistry struct {
	mu        sync.RWMutex
	pausables map[string]pausable.Pausable
}

func newServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		pausables: make(map[string]pausable.Pausable),
	}
}

// Register adds a Pausable service to the registry.
func (r *ServiceRegistry) Register(p pausable.Pausable) {
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
		return fmt.Errorf("%w: %q", ErrServiceNotFound, name)
	}
	return p.Pause()
}

// Resume resumes the named service. Returns an error if the service is not found.
func (r *ServiceRegistry) Resume(name string) error {
	r.mu.RLock()
	p, ok := r.pausables[name]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("%w: %q", ErrServiceNotFound, name)
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
