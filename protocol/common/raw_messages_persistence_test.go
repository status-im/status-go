package common

import (
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/crypto"
	messagingtypes "github.com/status-im/status-go/messaging/types"
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

	err = p.SaveRawMessage(&messagingtypes.RawMessage{
		ID:                    "1",
		ResendType:            messagingtypes.ResendTypeRawMessage,
		LocalChatID:           "",
		CommunityID:           []byte("c1"),
		CommunityKeyExMsgType: messagingtypes.KeyExMsgRekey,
		Sender:                pk,
		ResendMethod:          messagingtypes.ResendMethodSendPrivate,
		Recipients:            []*ecdsa.PublicKey{pk.Public().(*ecdsa.PublicKey)},
	})
	require.NoError(t, err)
	m, err := p.RawMessageByID("1")
	require.NoError(t, err)
	require.Equal(t, "1", m.ID)
	require.Equal(t, messagingtypes.ResendTypeRawMessage, m.ResendType)
	require.Equal(t, messagingtypes.KeyExMsgRekey, m.CommunityKeyExMsgType)
	require.Equal(t, "c1", string(m.CommunityID))
	require.Equal(t, pk, m.Sender)
	require.Equal(t, messagingtypes.ResendMethodSendPrivate, m.ResendMethod)
	require.Equal(t, 1, len(m.Recipients))
}

func TestUpdateRawMessageSent(t *testing.T) {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	require.NoError(t, sqlite.Migrate(db))
	p := NewRawMessagesPersistence(db)

	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	rawMessageID := "1"
	err = p.SaveRawMessage(buildRawMessage(rawMessageID, pk))
	require.NoError(t, err)

	rawMessage, err := p.RawMessageByID(rawMessageID)
	require.NoError(t, err)
	require.True(t, rawMessage.Sent)
	require.Greater(t, rawMessage.LastSent, uint64(0))

	err = p.UpdateRawMessageSent(rawMessageID, false)
	require.NoError(t, err)

	m, err := p.RawMessageByID(rawMessageID)
	require.NoError(t, err)
	require.False(t, m.Sent)
}

func TestUpdateRawMessageLastSent(t *testing.T) {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	require.NoError(t, sqlite.Migrate(db))
	p := NewRawMessagesPersistence(db)

	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	rawMessageID := "1"
	err = p.SaveRawMessage(buildRawMessage(rawMessageID, pk))
	require.NoError(t, err)

	rawMessage, err := p.RawMessageByID(rawMessageID)
	require.NoError(t, err)
	require.True(t, rawMessage.Sent)
	require.Greater(t, rawMessage.LastSent, uint64(0))

	err = p.UpdateRawMessageLastSent(rawMessageID, 0)
	require.NoError(t, err)

	m, err := p.RawMessageByID(rawMessageID)
	require.NoError(t, err)
	require.Equal(t, m.LastSent, uint64(0))
}

func buildRawMessage(rawMessageID string, pk *ecdsa.PrivateKey) *messagingtypes.RawMessage {
	return &messagingtypes.RawMessage{
		ID:                    rawMessageID,
		ResendType:            messagingtypes.ResendTypeRawMessage,
		LocalChatID:           "",
		CommunityID:           []byte("c1"),
		CommunityKeyExMsgType: messagingtypes.KeyExMsgRekey,
		Sender:                pk,
		ResendMethod:          messagingtypes.ResendMethodSendPrivate,
		Recipients:            []*ecdsa.PublicKey{pk.Public().(*ecdsa.PublicKey)},
		Sent:                  true,
		LastSent:              uint64(time.Now().UnixNano() / int64(time.Millisecond)),
	}
}
