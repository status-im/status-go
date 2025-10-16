package wakuv2

import (
	"testing"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/messaging/waku/migrations"
	"github.com/status-im/status-go/t/helpers"
)

func TestProtectedTopicsPersistence(t *testing.T) {
	db, err := helpers.SetupTestMemorySQLDB(helpers.NewTestDBInitializer(bindata.Resource(
		migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		},
	)))
	require.NoError(t, err)

	p := NewSQLiteProtectedTopicsPersistence(db)

	// Generate ECDSA keys
	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	pubKey := &privKey.PublicKey

	pubsubTopic := "test-topic"

	// Insert protected topic
	err = p.Insert(pubsubTopic, privKey, pubKey)
	require.NoError(t, err)

	// Fetch private key for topic
	fetchedPrivKey, err := p.FetchPrivateKey(pubsubTopic)
	require.NoError(t, err)
	require.NotNil(t, fetchedPrivKey)
	require.Equal(t, privKey.D.Bytes(), fetchedPrivKey.D.Bytes())

	// Fetch protected topics
	topics, err := p.All()
	require.NoError(t, err)
	require.Len(t, topics, 1)
	require.Equal(t, pubsubTopic, topics[0].Topic)

	// Delete protected topic
	err = p.Delete(pubsubTopic)
	require.NoError(t, err)

	// Ensure topic is deleted
	topics, err = p.All()
	require.NoError(t, err)
	require.Len(t, topics, 0)

	fetchedPrivKey, err = p.FetchPrivateKey(pubsubTopic)
	require.NoError(t, err)
	require.Nil(t, fetchedPrivKey)
}
