package server

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestServerURLSuite(t *testing.T) {
	suite.Run(t, new(ServerURLSuite))
}

type ServerURLSuite struct {
	suite.Suite
	TestKeyComponents

	server       *MediaServer
	serverNoPort *MediaServer
}

func (s *ServerURLSuite) SetupSuite() {
	s.SetupKeyComponents(s.T())

	s.server = &MediaServer{Server: Server{
		hostname:   defaultIP.String(),
		portManger: newPortManager(nil),
	}}
	err := s.server.SetPort(1337)
	s.Require().NoError(err)

	s.serverNoPort = &MediaServer{Server: Server{
		hostname: defaultIP.String(),
	}}
}

func (s *ServerURLSuite) TestServer_MakeBaseURL() {
	s.Require().Equal("https://127.0.0.1:1337", s.server.MakeBaseURL().String())
	s.Require().Equal("https://127.0.0.1:0", s.serverNoPort.MakeBaseURL().String())
}

func (s *ServerURLSuite) TestServer_MakeImageServerURL() {
	s.Require().Equal("https://127.0.0.1:1337/messages/", s.server.MakeImageServerURL())
	s.Require().Equal("https://127.0.0.1:0/messages/", s.serverNoPort.MakeImageServerURL())
}

func (s *ServerURLSuite) TestServer_MakeIdenticonURL() {
	s.Require().Equal(
		"https://127.0.0.1:1337/messages/identicons?publicKey=0xdaff0d11decade",
		s.server.MakeIdenticonURL("0xdaff0d11decade"))
	s.Require().Equal(
		"https://127.0.0.1:0/messages/identicons?publicKey=0xdaff0d11decade",
		s.serverNoPort.MakeIdenticonURL("0xdaff0d11decade"))
}

func (s *ServerURLSuite) TestServer_MakeImageURL() {
	s.Require().Equal(
		"https://127.0.0.1:1337/messages/images?messageId=0x10aded70ffee",
		s.server.MakeImageURL("0x10aded70ffee"))
	s.Require().Equal(
		"https://127.0.0.1:0/messages/images?messageId=0x10aded70ffee",
		s.serverNoPort.MakeImageURL("0x10aded70ffee"))
}

func (s *ServerURLSuite) TestServer_MakeAudioURL() {
	s.Require().Equal(
		"https://127.0.0.1:1337/messages/audio?messageId=0xde1e7ebee71e",
		s.server.MakeAudioURL("0xde1e7ebee71e"))
	s.Require().Equal(
		"https://127.0.0.1:0/messages/audio?messageId=0xde1e7ebee71e",
		s.serverNoPort.MakeAudioURL("0xde1e7ebee71e"))
}

func (s *ServerURLSuite) TestServer_MakeStickerURL() {
	s.Require().Equal(
		"https://127.0.0.1:1337/ipfs?hash=0xdeadbeef4ac0",
		s.server.MakeStickerURL("0xdeadbeef4ac0"))
	s.Require().Equal(
		"https://127.0.0.1:0/ipfs?hash=0xdeadbeef4ac0",
		s.serverNoPort.MakeStickerURL("0xdeadbeef4ac0"))
}
