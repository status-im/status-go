package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gorilla/websocket"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/signal"
)

func TestSignalsServer(t *testing.T) {
	server := NewSignalsServer()
	server.Setup()
	err := server.Listen("localhost:0")
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Stop(ctx)
	}()

	addr := server.Address()
	serverURLString := fmt.Sprintf("ws://%s", addr)
	serverURL, err := url.Parse(serverURLString)
	require.NoError(t, err)
	require.NotZero(t, serverURL.Port())

	connection, _, err := websocket.DefaultDialer.Dial(serverURLString+"/signals", nil)
	require.NoError(t, err)
	require.NotNil(t, connection)
	defer func() {
		err := connection.Close()
		require.NoError(t, err)
	}()

	sentEvent := signal.MessageDeliveredSignal{
		ChatID:    randomAlphabeticalString(t, 10),
		MessageID: randomAlphabeticalString(t, 10),
	}

	signal.SendMessageDelivered(sentEvent.ChatID, sentEvent.MessageID)

	messageType, data, err := connection.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, websocket.TextMessage, messageType)

	receivedSignal := signal.Envelope{}
	err = json.Unmarshal(data, &receivedSignal)
	require.NoError(t, err)
	require.Equal(t, signal.EventMesssageDelivered, receivedSignal.Type)
	require.NotNil(t, receivedSignal.Event)

	// Convert `interface{}` to json and then back to the original struct
	tempJson, err := json.Marshal(receivedSignal.Event)
	require.NoError(t, err)

	receivedEvent := signal.MessageDeliveredSignal{}
	err = json.Unmarshal(tempJson, &receivedEvent)
	require.Equal(t, sentEvent, receivedEvent)
}

func randomAlphabeticalString(t *testing.T, n int) string {
	s, err := common.RandomAlphabeticalString(n)
	require.NoError(t, err)
	return s
}
