package server

import (
	"fmt"
	"sync"
)

// portManager is responsible for maintaining segregated access to the port field via the use of portWait sync.Mutex
type portManger struct {
	port             int
	afterPortChanged func(port int)
	portWait         *sync.Mutex
}

// newPortManager returns a newly initialised portManager with a pre-Locked portManger.portWait sync.Mutex
func newPortManager(afterPortChanged func(int)) portManger {
	pm := portManger{
		afterPortChanged: afterPortChanged,
		portWait:         new(sync.Mutex),
	}
	pm.portWait.Lock()
	return pm
}

// SetPort sets the internal portManger.port field to the given port value
// next triggers any given portManger.afterPortChanged function
// additionally portManger.portWait.Unlock() is called, releasing any calls to MustGetPort
func (p *portManger) SetPort(port int) error {
	// TryLock, multiple portManager.SetPort calls trigger `fatal error: sync: unlock of unlocked mutex`
	// In the case of concurrent
	// TODO fix this horrible thing
	p.portWait.TryLock()

	if port == 0 {
		return fmt.Errorf("port can not be `0`, use ResetPort() instead")
	}

	p.port = port
	if p.afterPortChanged != nil {
		p.afterPortChanged(port)
	}
	p.portWait.Unlock()
	return nil
}

// ResetPort attempts to reset portManger.port to 0
// if portManger.portWait is already locked the function returns after doing nothing
func (p *portManger) ResetPort() {
	if p.portWait.TryLock() {
		p.port = 0
	}
}

// GetPort gets the current value of portManager.port without any concern for the state of its value
// and therefore does not block until portManager.portWait.Unlock() is called
func (p *portManger) GetPort() int {
	return p.port
}

// MustGetPort only returns portManager.port if portManager.portWait is unlocked.
// This presupposes that portManger.portWait has a default state of locked and SetPort unlock portManager.portWait
func (p *portManger) MustGetPort() int {
	p.portWait.Lock()
	defer p.portWait.Unlock()

	return p.port
}
