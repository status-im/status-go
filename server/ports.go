package server

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// portManager is responsible for maintaining segregated access to the port field
// via the use of rwLock sync.RWMutex and mustRead sync.Mutex
// rwLock establishes a standard read write mutex that allows consecutive reads and exclusive writes
// mustRead forces MustGetPort to wait until port has a none 0 value
type portManger struct {
	logger           *zap.Logger
	port             int
	afterPortChanged func(port int)
	rwLock           *sync.RWMutex
	mustRead         *sync.Mutex
}

// newPortManager returns a newly initialised portManager with a pre-Locked portManger.mustRead sync.Mutex
func newPortManager(logger *zap.Logger, afterPortChanged func(int)) portManger {
	pm := portManger{
		logger:           logger.Named("portManger"),
		afterPortChanged: afterPortChanged,
		rwLock:           new(sync.RWMutex),
		mustRead:         new(sync.Mutex),
	}
	pm.mustRead.Lock()
	return pm
}

// SetPort sets portManger.port field to the given port value
// next triggers any given portManger.afterPortChanged function
// additionally portManger.mustRead.Unlock() is called, releasing any calls to MustGetPort
func (p *portManger) SetPort(port int) error {
	l := p.logger.Named("SetPort")
	l.Debug("fired", zap.Int("port", port))

	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	l.Debug("acquired rwLock.Lock")

	if port == 0 {
		errMsg := "port can not be `0`, use ResetPort() instead"
		l.Error(errMsg)
		return fmt.Errorf(errMsg)
	}

	p.port = port
	if p.afterPortChanged != nil {
		l.Debug("p.afterPortChanged != nil")
		p.afterPortChanged(port)
	}
	p.mustRead.Unlock()
	l.Debug("p.mustRead.Unlock()")
	return nil
}

// ResetPort attempts to reset portManger.port to 0
// if portManger.mustRead is already locked the function returns after doing nothing
// portManger.mustRead.TryLock() is called because ResetPort may be called multiple times in a row
// and calling multiple times must not cause a deadlock or an infinite hang, but the lock needs to be
// reapplied if it is not present.
func (p *portManger) ResetPort() {
	l := p.logger.Named("ResetPort")
	l.Debug("fired")

	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	l.Debug("acquired rwLock.Lock")

	if p.mustRead.TryLock() {
		l.Debug("able to lock mustRead")
		p.port = 0
		return
	}
	l.Debug("unable to lock mustRead")
}

// GetPort gets the current value of portManager.port without any concern for the state of its value
// and therefore does not block until portManager.mustRead.Unlock() is called
func (p *portManger) GetPort() int {
	l := p.logger.Named("GetPort")
	l.Debug("fired")

	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	l.Debug("acquired rwLock.RLock")

	return p.port
}

// MustGetPort only returns portManager.port if portManager.mustRead is unlocked.
// This presupposes that portManger.mustRead has a default state of locked and SetPort unlock portManager.mustRead
func (p *portManger) MustGetPort() int {
	l := p.logger.Named("MustGetPort")
	l.Debug("fired")

	p.mustRead.Lock()
	defer p.mustRead.Unlock()
	l.Debug("acquired mustRead.Lock")

	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	l.Debug("acquired rwLock.RLock")

	return p.port
}
