package server

import (
	"fmt"
	"sync"
)

type portManger struct {
	port             int
	afterPortChanged func(port int)
	portWait         *sync.Mutex
}

func newPortManager(afterPortChanged func(int)) portManger {
	pm := portManger{
		afterPortChanged: afterPortChanged,
		portWait:         new(sync.Mutex),
	}
	pm.portWait.Lock()
	return pm
}

func (p *portManger) SetPort(port int) error {
	if port == 0 {
		return fmt.Errorf("port can not be `0`, use ResetPort() instead")
	}

	p.port = port
	p.portWait.Unlock()
	return nil
}

func (p *portManger) ResetPort() {
	if p.portWait.TryLock() {
		p.port = 0
	}
}

func (p *portManger) GetPort() int {
	return p.port
}

func (p *portManger) MustGetPort() int {
	p.portWait.Lock()
	defer p.portWait.Unlock()

	return p.port
}
