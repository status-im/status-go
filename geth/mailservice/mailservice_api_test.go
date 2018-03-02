package mailservice

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestRequestMessagesDefaults(t *testing.T) {
	r := MessagesRequest{}
	setMessagesRequestDefaults(&r)
	require.NotZero(t, r.From)
	require.InEpsilon(t, uint32(time.Now().UTC().Unix()), r.To, 1.0)
}

func TestRequestMessagesFailures(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := NewMockServiceProvider(ctrl)
	api := NewPublicAPI(provider)
	shh := whisper.New(nil)
	// Node is ephemeral (only in memory).
	nodeA, nodeErr := node.New(&node.Config{NoUSB: true})
	require.NoError(t, nodeErr)
	require.NoError(t, nodeA.Start())
	defer func() {
		err := nodeA.Stop()
		require.NoError(t, err)
	}()

	const (
		mailServerPeer = "enode://b7e65e1bedc2499ee6cbd806945af5e7df0e59e4070c96821570bd581473eade24a489f5ec95d060c0db118c879403ab88d827d3766978f28708989d35474f87@[::]:51920"
	)

	var (
		result bool
		err    error
	)

	// invalid MailServer enode address
	provider.EXPECT().WhisperService().Return(nil, nil)
	provider.EXPECT().Node().Return(nil, nil)
	result, err = api.RequestMessages(context.TODO(), MessagesRequest{MailServerPeer: "invalid-address"})
	require.False(t, result)
	require.EqualError(t, err, "invalid mailServerPeer value: invalid URL scheme, want \"enode\"")

	// non-existent symmetric key
	provider.EXPECT().WhisperService().Return(shh, nil)
	provider.EXPECT().Node().Return(nil, nil)
	result, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailServerPeer,
	})
	require.False(t, result)
	require.EqualError(t, err, "invalid symKeyID value: non-existent key ID")

	// with a symmetric key
	symKeyID, symKeyErr := shh.AddSymKeyFromPassword("some-pass")
	require.NoError(t, symKeyErr)
	provider.EXPECT().WhisperService().Return(shh, nil)
	provider.EXPECT().Node().Return(nodeA, nil)
	result, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailServerPeer,
		SymKeyID:       symKeyID,
	})
	require.Contains(t, err.Error(), "Could not find peer with ID")
	require.False(t, result)
}

func TestRequestMessagesSuccess(t *testing.T) {
	// TODO(adam): next step would be to run a successful test, however,
	// it requires to set up emepheral nodes that can discover each other
	// without syncing blockchain. It requires a bit research how to do that.
	t.Skip()
}
