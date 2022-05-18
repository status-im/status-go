package server

import (
	"net"
	"testing"

	"github.com/stretchr/testify/suite"
)

var (
	baseRegex = "https:\\/\\/127\\.0\\.0\\.1:[0-9]{2,5}"
)

func TestServerURLSuite(t *testing.T) {
	suite.Run(t, new(ServerURLSuite))
}

type ServerURLSuite struct {
	suite.Suite

	server           *MediaServer
	serverNoListener *MediaServer
}

func (s *ServerURLSuite) SetupSuite() {
	l, err := net.Listen("tcp", defaultIP.String()+":0")
	s.Require().NoError(err)

	s.server = &MediaServer{Server: Server{
		netIP:    defaultIP,
		listener: l,
	}}
	s.serverNoListener = &MediaServer{Server: Server{
		netIP: defaultIP,
	}}
}

func (s *ServerURLSuite) TestServer_MakeBaseURL() {
	s.Require().Regexp(baseRegex, s.server.MakeBaseURL().String())
	s.Require().Equal("https://127.0.0.1:0", s.serverNoListener.MakeBaseURL().String())
}

func (s *ServerURLSuite) TestServer_MakeImageServerURL() {
	s.Require().Regexp(baseRegex+"\\/messages\\/", s.server.MakeImageServerURL())
	s.Require().Equal("https://127.0.0.1:0/messages/", s.serverNoListener.MakeImageServerURL())
}

func (s *ServerURLSuite) TestServer_MakeIdenticonURL() {
	s.Require().Regexp(
		baseRegex+"\\/messages\\/identicons\\?publicKey=0xdaff0d11decade",
		s.server.MakeIdenticonURL("0xdaff0d11decade"))
	s.Require().Equal(
		"https://127.0.0.1:0/messages/identicons?publicKey=0xdaff0d11decade",
		s.serverNoListener.MakeIdenticonURL("0xdaff0d11decade"))
}

func (s *ServerURLSuite) TestServer_MakeImageURL() {
	s.Require().Regexp(
		baseRegex+"\\/messages\\/images\\?messageId=0x10aded70ffee",
		s.server.MakeImageURL("0x10aded70ffee"))
	s.Require().Equal(
		"https://127.0.0.1:0/messages/images?messageId=0x10aded70ffee",
		s.serverNoListener.MakeImageURL("0x10aded70ffee"))
}

func (s *ServerURLSuite) TestServer_MakeAudioURL() {
	s.Require().Regexp(
		baseRegex+"\\/messages\\/audio\\?messageId=0xde1e7ebee71e",
		s.server.MakeAudioURL("0xde1e7ebee71e"))
	s.Require().Equal(
		"https://127.0.0.1:0/messages/audio?messageId=0xde1e7ebee71e",
		s.serverNoListener.MakeAudioURL("0xde1e7ebee71e"))
}

func (s *ServerURLSuite) TestServer_MakeStickerURL() {
	s.Require().Regexp(
		baseRegex+"\\/ipfs\\?hash=0xdeadbeef4ac0",
		s.server.MakeStickerURL("0xdeadbeef4ac0"))
	s.Require().Equal(
		"https://127.0.0.1:0/ipfs?hash=0xdeadbeef4ac0",
		s.serverNoListener.MakeStickerURL("0xdeadbeef4ac0"))
}
