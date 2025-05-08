package transport

import (
	"testing"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"

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
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	err = sqlite.Migrate(db)
	require.NoError(t, err)

	require.NoError(t, err)

	logger := tt.MustCreateTestLogger()
	require.NoError(t, err)
	defer func() { _ = logger.Sync() }()

	_, err = NewTransport(nil, nil, &keysPersistenceMock{}, &processedMessageIDsCacheMock{}, nil, logger)
	require.NoError(t, err)
}
