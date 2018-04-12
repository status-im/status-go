package mailservice

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/stretchr/testify/require"
)

func TestRequestMessagesDefaults(t *testing.T) {
	r := MessagesRequest{}
	setMessagesRequestDefaults(&r)
	require.NotZero(t, r.From)
	require.InEpsilon(t, uint32(time.Now().UTC().Unix()), r.To, 1.0)
}

func TestRequestMessages(t *testing.T) {
	var err error

	shh := whisper.New(nil)
	aNode, err := node.New(&node.Config{
		NoUSB: true,
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
		},
	}) // in-memory node as no data dir
	require.NoError(t, err)
	err = aNode.Register(func(_ *node.ServiceContext) (node.Service, error) {
		return shh, nil
	})
	require.NoError(t, err)

	err = aNode.Start()
	require.NoError(t, err)
	defer func() {
		err := aNode.Stop()
		require.NoError(t, err)
	}()

	service := New(aNode, shh)
	api := NewPublicAPI(service)

	const (
		mailServerPeer = "enode://b7e65e1bedc2499ee6cbd806945af5e7df0e59e4070c96821570bd581473eade24a489f5ec95d060c0db118c879403ab88d827d3766978f28708989d35474f87@[::]:51920"
	)

	var result bool

	// invalid MailServer enode address
	result, err = api.RequestMessages(context.TODO(), MessagesRequest{MailServerPeer: "invalid-address"})
	require.False(t, result)
	require.EqualError(t, err, "invalid mailServerPeer value: invalid URL scheme, want \"enode\"")

	// non-existent symmetric key
	result, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailServerPeer,
	})
	require.False(t, result)
	require.EqualError(t, err, "invalid symKeyID value: non-existent key ID")

	// with a symmetric key
	symKeyID, symKeyErr := shh.AddSymKeyFromPassword("some-pass")
	require.NoError(t, symKeyErr)
	result, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailServerPeer,
		SymKeyID:       symKeyID,
	})
	require.Contains(t, err.Error(), "Could not find peer with ID")
	require.False(t, result)

	// with a peer acting line a mailserver
	// prepare a node first
	mailNode, err := node.New(&node.Config{
		NoUSB: true,
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
			ListenAddr:  ":0",
		},
	}) // in-memory node as no data dir
	require.NoError(t, err)
	err = mailNode.Register(func(_ *node.ServiceContext) (node.Service, error) {
		return whisper.New(nil), nil
	})
	require.NoError(t, err)
	err = mailNode.Start()
	require.NoError(t, err)
	defer func() {
		err := mailNode.Stop()
		require.NoError(t, err)
	}()

	// add mailPeer as a peer
	aNode.Server().AddPeer(mailNode.Server().Self())
	time.Sleep(time.Second * 10)

	// send a request
	result, err = api.RequestMessages(context.TODO(), MessagesRequest{
		MailServerPeer: mailNode.Server().Self().String(),
		SymKeyID:       symKeyID,
	})
	require.NoError(t, err)
	require.True(t, result)
}
