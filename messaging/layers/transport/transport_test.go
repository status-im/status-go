package transport

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/tt"
)

type keysPersistenceMock struct {
}

func (p *keysPersistenceMock) All() (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

func (p *keysPersistenceMock) Add(chatID string, key []byte) error {
	return nil
}

type processedMessageIDsCacheMock struct {
}

func (p *processedMessageIDsCacheMock) Clear() error {
	return nil
}
func (p *processedMessageIDsCacheMock) Hits(ids []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (p *processedMessageIDsCacheMock) Add(ids []string, timestamp uint64) error {
	return nil
}
func (p *processedMessageIDsCacheMock) Clean(timestamp uint64) error {
	return nil
}

func TestNewTransport(t *testing.T) {
	logger := tt.MustCreateTestLogger()
	defer func() { _ = logger.Sync() }()

	_, err := NewTransport(nil, nil, &keysPersistenceMock{}, &processedMessageIDsCacheMock{}, nil, logger)
	require.NoError(t, err)
}
