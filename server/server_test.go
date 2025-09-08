package server

import (
	"fmt"
	"net"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/common"
)

func TestServerURLSuite(t *testing.T) {
	suite.Run(t, new(ServerURLSuite))
}

type ServerURLSuite struct {
	suite.Suite

	baseURL string
	server  *MediaServer
}

func (s *ServerURLSuite) SetupTest() {
	port := gofakeit.Number(1000, 65535)
	ip := gofakeit.IPv4Address()
	s.baseURL = fmt.Sprintf("https://%s:%d", ip, port)

	s.server = &MediaServer{
		Server: Server{
			address: &net.TCPAddr{
				IP:   net.ParseIP(ip),
				Port: port,
			},
		},
	}
}

func (s *ServerURLSuite) TestServer_MakeBaseURL() {
	s.Require().Equal(s.baseURL, s.server.MakeBaseURL().String())
}

func (s *ServerURLSuite) TestServer_MakeImageServerURL() {
	s.Require().Equal(s.baseURL+"/messages/", s.server.MakeImageServerURL())
}

func (s *ServerURLSuite) TestServer_MakeImageURL() {
	s.Require().Equal(
		s.baseURL+"/messages/images?messageId=0x10aded70ffee",
		s.server.MakeImageURL("0x10aded70ffee"))
}

func (s *ServerURLSuite) TestServer_MakeLinkPreviewThumbnailURL() {
	s.Require().Equal(
		s.baseURL+"/link-preview/thumbnail?message-id=99&url=https%3A%2F%2Fgithub.com",
		s.server.MakeLinkPreviewThumbnailURL("99", "https://github.com"))
}

func (s *ServerURLSuite) TestServer_MakeStatusLinkPreviewThumbnailURL() {
	s.Require().Equal(
		s.baseURL+"/status-link-preview/thumbnail?image-id=contact-icon&message-id=99&url=https%3A%2F%2Fstatus.app",
		s.server.MakeStatusLinkPreviewThumbnailURL("99", "https://status.app", common.MediaServerContactIcon))
}

func (s *ServerURLSuite) TestServer_MakeAudioURL() {
	s.Require().Equal(
		s.baseURL+"/messages/audio?messageId=0xde1e7ebee71e",
		s.server.MakeAudioURL("0xde1e7ebee71e"))
}

func (s *ServerURLSuite) TestServer_MakeStickerURL() {
	s.Require().Equal(
		s.baseURL+"/ipfs?hash=0xdeadbeef4ac0",
		s.server.MakeStickerURL("0xdeadbeef4ac0"))
}

func (s *ServerURLSuite) TestServer_MakeContactImageURL() {
	s.Require().Equal(
		s.baseURL+"/contactImages?clock=1&imageName=Test&publicKey=0x1",
		s.server.MakeContactImageURL("0x1", "Test", uint64(1)))
}
