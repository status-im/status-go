package server

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func setupServerForURLTests() *Server {
	return &Server{
		port:   1337,
		netIp:  defaultIp,
	}
}

func TestServer_MakeBaseURL(t *testing.T) {
	s := setupServerForURLTests()

	require.Equal(t, "https://127.0.0.1:1337", s.MakeBaseURL().String())
}

func TestServer_MakeImageServerURL(t *testing.T) {
	s := setupServerForURLTests()

	require.Equal(t, "https://127.0.0.1:1337/messages/", s.MakeImageServerURL())
}

func TestServer_MakeIdenticonURL(t *testing.T) {
	s := setupServerForURLTests()

	require.Equal(t, "https://127.0.0.1:1337/messages/identicons?publicKey=0xdaff0d11decade", s.MakeIdenticonURL("0xdaff0d11decade"))
}

func TestServer_MakeImageURL(t *testing.T) {
	s := setupServerForURLTests()

	require.Equal(t, "https://127.0.0.1:1337/messages/images?messageId=0x10aded70ffee", s.MakeImageURL("0x10aded70ffee"))
}

func TestServer_MakeAudioURL(t *testing.T) {
	s := setupServerForURLTests()

	require.Equal(t, "https://127.0.0.1:1337/messages/audio?messageId=0xde1e7ebee71e", s.MakeAudioURL("0xde1e7ebee71e"))
}
