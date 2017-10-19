package node

import (
	"github.com/status-im/status-go/geth/common/services"
	"sync"
)

type les struct {
	l    services.LesService // reference to LES service
	back services.StatusBackend
	*sync.RWMutex
}

func newLES() *les {
	m := &sync.RWMutex{}
	return &les{RWMutex: m}
}
