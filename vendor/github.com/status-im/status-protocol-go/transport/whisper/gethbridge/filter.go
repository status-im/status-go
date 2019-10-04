package gethbridge

import (
	"crypto/ecdsa"

	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
	whisper "github.com/status-im/whisper/whisperv6"
)

type gethFilterWrapper struct {
	filter *whisper.Filter
}

// NewGethFilterWrapper returns an object that wraps Geth's Filter in a whispertypes interface
func NewGethFilterWrapper(f *whisper.Filter) whispertypes.Filter {
	if f.Messages == nil {
		panic("Messages should not be nil")
	}

	return &gethFilterWrapper{
		filter: f,
	}
}

// GetGethFilterFrom retrieves the underlying whisper Filter struct from a wrapped Filter interface
func GetGethFilterFrom(f whispertypes.Filter) *whisper.Filter {
	return f.(*gethFilterWrapper).filter
}

// KeyAsym returns the private Key of recipient
func (w *gethFilterWrapper) KeyAsym() *ecdsa.PrivateKey {
	return w.filter.KeyAsym
}

// KeySym returns the key associated with the Topic
func (w *gethFilterWrapper) KeySym() []byte {
	return w.filter.KeySym
}
