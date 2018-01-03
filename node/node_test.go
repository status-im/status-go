package node

import (
	"testing"
	"time"

	"github.com/status-im/status-go/e2e"
	"github.com/status-im/status-go/geth/params"
	"github.com/stretchr/testify/require"
)

func nodeConfigTest() *params.NodeConfig {
	return &params.NodeConfig{
		DevMode:         true,
		NetworkID:       params.RopstenNetworkID,
		DataDir:         "./test-data",
		Name:            params.ClientIdentifier,
		Version:         "0.0.0.0",
		RPCEnabled:      params.RPCEnabledDefault,
		HTTPHost:        params.HTTPHost,
		HTTPPort:        params.HTTPPort,
		ListenAddr:      params.ListenAddr,
		APIModules:      params.APIModules,
		WSHost:          params.WSHost,
		WSPort:          params.WSPort,
		MaxPeers:        params.MaxPeers,
		MaxPendingPeers: params.MaxPendingPeers,
		IPCFile:         params.IPCFile,
		LogFile:         params.LogFile,
		LogLevel:        params.LogLevel,
		LogToStderr:     params.LogToStderr,
		BootClusterConfig: &params.BootClusterConfig{
			Enabled:   true,
			BootNodes: []string{},
		},
		LightEthConfig: &params.LightEthConfig{
			Enabled:       true,
			DatabaseCache: params.DatabaseCache,
		},
		WhisperConfig: &params.WhisperConfig{
			Enabled:    true,
			MinimumPoW: params.WhisperMinimumPoW,
			TTL:        params.WhisperTTL,
			FirebaseConfig: &params.FirebaseConfig{
				NotificationTriggerURL: params.FirebaseNotificationTriggerURL,
			},
		},
		SwarmConfig: &params.SwarmConfig{},
	}
}

func newTestNode(t *testing.T) *StatusNode {
	config, err := e2e.MakeTestNodeConfig(params.RopstenNetworkID)
	require.Nil(t, err)
	sn, err := New(config)
	require.Nil(t, err)
	require.NotNil(t, sn)

	return sn
}

func TestNew(t *testing.T) {
	sn := newTestNode(t)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("error closing `Started` channel: %+v", r)
		}
	}()

	close(sn.Started)
}

func TestNode_Start_Stop(t *testing.T) {
	sn := newTestNode(t)
	sn.Start()

	waitFor := time.Duration(200)

	select {
	case <-sn.Started:
		t.Log("node started")
	case <-time.After(time.Millisecond * waitFor):
		t.Fatalf("node hasn't started after %d milliseconds", waitFor)
	}
}
