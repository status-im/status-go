package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
	messagingtypes "github.com/status-im/status-go/messaging/types"
)

func TestConfirmations(t *testing.T) {
	dataSyncID1 := []byte("datsync-id-1")
	dataSyncID2 := []byte("datsync-id-2")
	dataSyncID3 := []byte("datsync-id-3")
	dataSyncID4 := []byte("datsync-id-3")

	messageID1 := []byte("message-id-1")
	messageID2 := []byte("message-id-2")

	publicKey1 := []byte("pk-1")
	publicKey2 := []byte("pk-2")
	publicKey3 := []byte("pk-3")

	db, err := openTestDB()
	require.NoError(t, err)
	p := NewMessagingPersistence(db)

	confirmation1 := &messagingtypes.RawMessageConfirmation{
		DataSyncID: dataSyncID1,
		MessageID:  messageID1,
		PublicKey:  publicKey1,
	}

	// Same datasyncID and same messageID, different pubkey
	confirmation2 := &messagingtypes.RawMessageConfirmation{
		DataSyncID: dataSyncID2,
		MessageID:  messageID1,
		PublicKey:  publicKey2,
	}

	// Different datasyncID and same messageID, different pubkey
	confirmation3 := &messagingtypes.RawMessageConfirmation{
		DataSyncID: dataSyncID3,
		MessageID:  messageID1,
		PublicKey:  publicKey3,
	}

	// Same dataSyncID, different messageID
	confirmation4 := &messagingtypes.RawMessageConfirmation{
		DataSyncID: dataSyncID4,
		MessageID:  messageID2,
		PublicKey:  publicKey1,
	}

	require.NoError(t, p.InsertPendingConfirmation(confirmation1))
	require.NoError(t, p.InsertPendingConfirmation(confirmation2))
	require.NoError(t, p.InsertPendingConfirmation(confirmation3))
	require.NoError(t, p.InsertPendingConfirmation(confirmation4))

	// We confirm the first datasync message, no confirmations
	messageID, err := p.MarkAsConfirmed(dataSyncID1, false)
	require.NoError(t, err)
	require.Nil(t, messageID)

	// We confirm the second datasync message, no confirmations
	messageID, err = p.MarkAsConfirmed(dataSyncID2, false)
	require.NoError(t, err)
	require.Nil(t, messageID)

	// We confirm the third datasync message, messageID1 should be confirmed
	messageID, err = p.MarkAsConfirmed(dataSyncID3, false)
	require.NoError(t, err)
	require.Equal(t, messageID, types.HexBytes(messageID1))
}

func TestConfirmationsAtLeastOne(t *testing.T) {
	dataSyncID1 := []byte("datsync-id-1")
	dataSyncID2 := []byte("datsync-id-2")
	dataSyncID3 := []byte("datsync-id-3")

	messageID1 := []byte("message-id-1")

	publicKey1 := []byte("pk-1")
	publicKey2 := []byte("pk-2")
	publicKey3 := []byte("pk-3")

	db, err := openTestDB()
	require.NoError(t, err)
	p := NewMessagingPersistence(db)

	confirmation1 := &messagingtypes.RawMessageConfirmation{
		DataSyncID: dataSyncID1,
		MessageID:  messageID1,
		PublicKey:  publicKey1,
	}

	// Same datasyncID and same messageID, different pubkey
	confirmation2 := &messagingtypes.RawMessageConfirmation{
		DataSyncID: dataSyncID2,
		MessageID:  messageID1,
		PublicKey:  publicKey2,
	}

	// Different datasyncID and same messageID, different pubkey
	confirmation3 := &messagingtypes.RawMessageConfirmation{
		DataSyncID: dataSyncID3,
		MessageID:  messageID1,
		PublicKey:  publicKey3,
	}

	require.NoError(t, p.InsertPendingConfirmation(confirmation1))
	require.NoError(t, p.InsertPendingConfirmation(confirmation2))
	require.NoError(t, p.InsertPendingConfirmation(confirmation3))

	// We confirm the first datasync message, messageID1 and 3 should be confirmed
	messageID, err := p.MarkAsConfirmed(dataSyncID1, true)
	require.NoError(t, err)
	require.NotNil(t, messageID)
	require.Equal(t, types.HexBytes(messageID1), messageID)
}

func TestSaveHashRatchetMessage(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := NewMessagingPersistence(db)

	groupID1 := []byte("group-id-1")
	groupID2 := []byte("group-id-2")
	keyID := []byte("key-id")

	message1 := &messagingtypes.ReceivedMessage{
		Hash:      []byte{1},
		Sig:       []byte{2},
		Timestamp: 2,
		Payload:   []byte{3},
	}

	require.NoError(t, p.SaveHashRatchetMessage(groupID1, keyID, message1))

	message2 := &messagingtypes.ReceivedMessage{
		Hash:      []byte{2},
		Sig:       []byte{2},
		Topic:     messagingtypes.BytesToContentTopic([]byte{5}),
		Timestamp: 2,
		Payload:   []byte{3},
		Dst:       []byte{4},
	}

	require.NoError(t, p.SaveHashRatchetMessage(groupID2, keyID, message2))

	fetchedMessages, err := p.GetHashRatchetMessages(keyID)
	require.NoError(t, err)
	require.NotNil(t, fetchedMessages)
	require.Len(t, fetchedMessages, 2)
}

func TestDeleteHashRatchetMessage(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := NewMessagingPersistence(db)

	groupID := []byte("group-id")
	keyID := []byte("key-id")

	message1 := &messagingtypes.ReceivedMessage{
		Hash:      []byte{1},
		Sig:       []byte{2},
		Timestamp: 2,
		Payload:   []byte{3},
	}

	require.NoError(t, p.SaveHashRatchetMessage(groupID, keyID, message1))

	message2 := &messagingtypes.ReceivedMessage{
		Hash:      []byte{2},
		Sig:       []byte{2},
		Topic:     messagingtypes.BytesToContentTopic([]byte{5}),
		Timestamp: 2,
		Payload:   []byte{3},
		Dst:       []byte{4},
	}

	require.NoError(t, p.SaveHashRatchetMessage(groupID, keyID, message2))

	message3 := &messagingtypes.ReceivedMessage{
		Hash:      []byte{3},
		Sig:       []byte{2},
		Topic:     messagingtypes.BytesToContentTopic([]byte{5}),
		Timestamp: 2,
		Payload:   []byte{3},
		Dst:       []byte{4},
	}

	require.NoError(t, p.SaveHashRatchetMessage(groupID, keyID, message3))

	fetchedMessages, err := p.GetHashRatchetMessages(keyID)
	require.NoError(t, err)
	require.NotNil(t, fetchedMessages)
	require.Len(t, fetchedMessages, 3)

	require.NoError(t, p.DeleteHashRatchetMessages([][]byte{[]byte{1}, []byte{2}}))

	fetchedMessages, err = p.GetHashRatchetMessages(keyID)
	require.NoError(t, err)
	require.NotNil(t, fetchedMessages)
	require.Len(t, fetchedMessages, 1)
}

func TestWakuProtectedTopicPersistence(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := NewMessagingPersistence(db)

	// Generate ECDSA keys
	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	pubKey := &privKey.PublicKey

	pubsubTopic := "test-topic"

	// Insert protected topic
	err = p.WakuStorage().InsertProtectedTopic(pubsubTopic, privKey, pubKey)
	require.NoError(t, err)

	// Fetch private key for topic
	fetchedPrivKey, err := p.WakuStorage().FetchPrivateKeyForProtectedTopic(pubsubTopic)
	require.NoError(t, err)
	require.NotNil(t, fetchedPrivKey)
	require.Equal(t, privKey.D.Bytes(), fetchedPrivKey.D.Bytes())

	// Fetch protected topics
	topics, err := p.WakuStorage().ProtectedTopics()
	require.NoError(t, err)
	require.Len(t, topics, 1)
	require.Equal(t, pubsubTopic, topics[0].Topic)

	// Delete protected topic
	err = p.WakuStorage().DeleteProtectedTopic(pubsubTopic)
	require.NoError(t, err)

	// Ensure topic is deleted
	topics, err = p.WakuStorage().ProtectedTopics()
	require.NoError(t, err)
	require.Len(t, topics, 0)

	fetchedPrivKey, err = p.WakuStorage().FetchPrivateKeyForProtectedTopic(pubsubTopic)
	require.NoError(t, err)
	require.Nil(t, fetchedPrivKey)
}
