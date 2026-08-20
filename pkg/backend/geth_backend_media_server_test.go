package backend

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/internal/testutils"
)

func reserveLocalhostPort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	return port
}

// TestMediaServerPortIgnoredOnRepeatedInitialize reproduces the client flow:
// login -> logout -> InitializeApplication -> login again.
//
// Without restarting the media server when InitializeApplication re-applies its
// options, the second login serves images on a random port even though
// InitializeApplication passes the same port.
func TestMediaServerPortIgnoredOnRepeatedInitialize(t *testing.T) {
	tmpdir := t.TempDir()
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	b.UpdateRootDataDir(tmpdir)

	port := reserveLocalhostPort(t)

	// InitializeApplication #1 + first login
	initializeApplicationWithMediaServerPortHelper(t, b, port)

	acc, err := b.CreateAccountAndLogin(&requests.CreateAccount{
		DisplayName:        "media-server-test",
		CustomizationColor: "#ffffff",
		Password:           testPassword,
		RootDataDir:        tmpdir,
		LogFilePath:        tmpdir + "/log",
		KdfIterations:      1,
	})
	require.NoError(t, err)
	require.Contains(t, b.StatusNode().MediaServer().GetListeningAddrPort(), fmt.Sprintf(":%d", port))

	// "login again": logout then InitializeApplication then login
	require.NoError(t, b.Logout())

	initializeApplicationWithMediaServerPortHelper(t, b, port)

	require.NoError(t, b.LoginAccount(&requests.Login{
		KeyUID:        acc.KeyUID,
		Password:      testPassword,
		KdfIterations: 1,
	}))

	require.Contains(t, b.StatusNode().MediaServer().GetListeningAddrPort(), fmt.Sprintf(":%d", port),
		"media server should keep the configured port after logout + re-initialize + login")
}

func initializeApplicationWithMediaServerPortHelper(t *testing.T, b *StatusBackend, port int) {
	t.Helper()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	enableTLS := false
	b.StatusNode().SetMediaServerOptions(&addr, &enableTLS, "localhost", port)
	require.NoError(t, b.OpenAccounts())
}
