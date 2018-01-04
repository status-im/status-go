package node

import (
	"testing"
	"time"

	"github.com/status-im/status-go/e2e"
	"github.com/status-im/status-go/geth/params"
	"github.com/stretchr/testify/require"
)

func newTestNode(t *testing.T) Node {
	config, err := e2e.MakeTestNodeConfig(params.RopstenNetworkID)
	require.Nil(t, err)
	sn, err := New(config)
	require.Nil(t, err)
	require.NotNil(t, sn)

	return sn
}

func TestNode_Start(t *testing.T) {
	sn := newTestNode(t)
	started, err := sn.Start()
	require.Nil(t, err)

	waitFor := time.Duration(200)

	select {
	case <-started:
		t.Log("node started")
	case <-time.After(time.Millisecond * waitFor):
		t.Fatalf("node hasn't started after %d milliseconds", waitFor)
	}
}
