package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"
)

func TestSaveRawMessage(t *testing.T) {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	require.NoError(t, sqlite.Migrate(db))
	p := NewRawMessagesPersistence(db)

	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	err = p.SaveRawMessage(&RawMessage{
		ID:                    "1",
		ResendType:            ResendTypeRawMessage,
		LocalChatID:           "",
		CommunityID:           []byte("c1"),
		CommunityKeyExMsgType: KeyExMsgRekey,
		Sender:                pk,
		ResendMethod:          ResendMethodSendPrivate,
	})
	require.NoError(t, err)
	m, err := p.RawMessageByID("1")
	require.NoError(t, err)
	require.Equal(t, "1", m.ID)
	require.Equal(t, ResendTypeRawMessage, m.ResendType)
	require.Equal(t, KeyExMsgRekey, m.CommunityKeyExMsgType)
	require.Equal(t, "c1", string(m.CommunityID))
	require.Equal(t, pk, m.Sender)
	require.Equal(t, ResendMethodSendPrivate, m.ResendMethod)
}
