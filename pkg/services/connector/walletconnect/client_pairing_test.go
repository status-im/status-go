package walletconnect

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestClient_Pair_ResurfacesProposalAfterDismissedModal(t *testing.T) {
	_, relay, client := newMockClient(t)

	relay.EXPECT().SetMessageHandler(gomock.Any()).Times(2)
	relay.EXPECT().Connect().Return(nil).Times(2)
	relay.EXPECT().Subscribe(testPairingTopic).Return("sub-id", nil).Times(2)
	relay.EXPECT().FetchMessages(testPairingTopic).Return(nil, false, fmt.Errorf("no messages")).Times(2)

	var delivered int32
	client.SetSessionProposalHandler(func(string) { atomic.AddInt32(&delivered, 1) })

	proposal, _ := json.Marshal(map[string]any{
		"id":     int64(123),
		"method": "wc_sessionPropose",
		"params": map[string]any{"proposer": map[string]any{"publicKey": "abcd1234"}},
	})
	encrypted, err := EncryptType0Envelope(testPairingSymKey, proposal)
	require.NoError(t, err)

	uri := validWCURI(testPairingTopic, testPairingSymKey)

	require.NoError(t, client.Pair(context.Background(), uri))
	client.handleRelayMessage(testPairingTopic, encrypted, tagSessionPropose)
	require.Eventually(t, func() bool { return atomic.LoadInt32(&delivered) == 1 },
		time.Second, 5*time.Millisecond, "first proposal was not delivered")

	// The user dismisses the modal, then pastes the same URI again.
	require.NoError(t, client.Pair(context.Background(), uri))
	client.handleRelayMessage(testPairingTopic, encrypted, tagSessionPropose)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&delivered) == 2 },
		time.Second, 5*time.Millisecond, "proposal was swallowed as a duplicate after re-pairing")
}
