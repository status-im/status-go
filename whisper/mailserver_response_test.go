package whisper

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func checkValidErrorPayload(t *testing.T, id []byte, errorMsg string) {
	requestID := common.BytesToHash(id)
	errPayload := CreateMailServerRequestFailedPayload(requestID, errors.New(errorMsg))
	nid := enode.ID{1}
	event, err := CreateMailServerEvent(nid, errPayload)

	require.NoError(t, err)
	require.NotNil(t, event)
	require.Equal(t, nid, event.Peer)
	require.Equal(t, requestID, event.Hash)

	eventData, ok := event.Data.(*MailServerResponse)
	if !ok {
		require.FailNow(t, "Unexpected data in event: %v, expected a MailServerResponse", event.Data)
	}
	require.EqualError(t, eventData.Error, errorMsg)
}

func checkValidSuccessPayload(t *testing.T, id []byte, lastHash []byte, timestamp uint32, envHash []byte) {
	requestID := common.BytesToHash(id)
	lastEnvelopeHash := common.BytesToHash(lastHash)
	timestampBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(timestampBytes, timestamp)
	envelopeHash := common.BytesToHash(envHash)
	cursor := append(timestampBytes, envelopeHash[:]...)
	successPayload := CreateMailServerRequestCompletedPayload(common.BytesToHash(id), lastEnvelopeHash, cursor)
	nid := enode.ID{1}
	event, err := CreateMailServerEvent(nid, successPayload)

	require.NoError(t, err)
	require.NotNil(t, event)
	require.Equal(t, nid, event.Peer)
	require.Equal(t, requestID, event.Hash)

	eventData, ok := event.Data.(*MailServerResponse)
	if !ok {
		require.FailNow(t, "Unexpected data in event: %v, expected a MailServerResponse", event.Data)
	}
	require.Equal(t, lastEnvelopeHash, eventData.LastEnvelopeHash)
	require.Equal(t, cursor, eventData.Cursor)
	require.NoError(t, eventData.Error)
}

func TestCreateMailServerEvent(t *testing.T) {
	// valid cases
	longErrorMessage := "longMessage|"
	for i := 0; i < 5; i++ {
		longErrorMessage = longErrorMessage + longErrorMessage
	}
	checkValidErrorPayload(t, []byte{0x01}, "test error 1")
	checkValidErrorPayload(t, []byte{0x02}, "test error 2")
	checkValidErrorPayload(t, []byte{0x02}, "")
	checkValidErrorPayload(t, []byte{0x00}, "test error 3")
	checkValidErrorPayload(t, []byte{}, "test error 4")

	checkValidSuccessPayload(t, []byte{0x01}, []byte{0x02}, 123, []byte{0x03})
	// invalid payloads

	// too small
	_, err := CreateMailServerEvent(enode.ID{}, []byte{0x00})
	require.Error(t, err)

	// too big and not error payload
	payloadTooBig := make([]byte, common.HashLength*2+cursorSize+100)
	_, err = CreateMailServerEvent(enode.ID{}, payloadTooBig)
	require.Error(t, err)
}
