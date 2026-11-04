package library

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStartFailureLeavesInstanceStopped checks that a failed Start() leaves the
// instance not-started and Free() safe (here the listen port is already in use).
func TestStartFailureLeavesInstanceStopped(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	addr := ln.Addr().(*net.TCPAddr)

	instance := Init()
	cfg := fmt.Sprintf(`{"host":"127.0.0.1","port":%d,"relay":false,"store":false,"discV5":false}`, addr.Port)
	require.NoError(t, NewNode(instance, cfg))

	err = Start(instance)
	require.Error(t, err, "Start should fail when the port is in use")
	require.False(t, IsStarted(instance), "instance must not report started after a failed Start")
	require.NotPanics(t, func() { _ = Free(instance) })
}
