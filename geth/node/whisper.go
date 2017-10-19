package node

import (
	"github.com/status-im/status-go/geth/common/services"
	"sync"
)

type whisper struct {
	w   services.Whisper
	api services.WhisperAPI
	*sync.RWMutex
}

func newWhisper() *whisper {
	m := &sync.RWMutex{}
	return &whisper{RWMutex: m}
}
