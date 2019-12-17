package gethbridge

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/whisper/v6"
)

type gethFilterWrapper struct {
	filter *whisper.Filter
	id     string
}

// NewGethFilterWrapper returns an object that wraps Geth's Filter in a types interface
func NewGethFilterWrapper(f *whisper.Filter, id string) types.Filter {
	if f.Messages == nil {
		panic("Messages should not be nil")
	}

	return &gethFilterWrapper{
		filter: f,
		id:     id,
	}
}

// GetGethFilterFrom retrieves the underlying whisper Filter struct from a wrapped Filter interface
func GetGethFilterFrom(f types.Filter) *whisper.Filter {
	return f.(*gethFilterWrapper).filter
}

// ID returns the filter ID
func (w *gethFilterWrapper) ID() string {
	return w.id
}
