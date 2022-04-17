package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	urlTestsServer = &Server{
		port:  1337,
		netIP: defaultIP,
	}
)

func TestServer_MakeBaseURL(t *testing.T) {
	require.Equal(t, "https://127.0.0.1:1337", urlTestsServer.MakeBaseURL().String())
}

func TestServer_MakeImageServerURL(t *testing.T) {
	require.Equal(t, "https://127.0.0.1:1337/messages/", urlTestsServer.MakeImageServerURL())
}

func TestServer_MakeIdenticonURL(t *testing.T) {
	require.Equal(t, "https://127.0.0.1:1337/messages/identicons?publicKey=0xdaff0d11decade", urlTestsServer.MakeIdenticonURL("0xdaff0d11decade"))
}

func TestServer_MakeImageURL(t *testing.T) {
	require.Equal(t, "https://127.0.0.1:1337/messages/images?messageId=0x10aded70ffee", urlTestsServer.MakeImageURL("0x10aded70ffee"))
}

func TestServer_MakeAudioURL(t *testing.T) {
	require.Equal(t, "https://127.0.0.1:1337/messages/audio?messageId=0xde1e7ebee71e", urlTestsServer.MakeAudioURL("0xde1e7ebee71e"))
}
