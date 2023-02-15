package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

const (
	waitTime = 50 * time.Millisecond
)

func TestServerURLSuite(t *testing.T) {
	suite.Run(t, new(ServerURLSuite))
}

type ServerURLSuite struct {
	suite.Suite
	TestKeyComponents
	TestLoggerComponents

	server       *MediaServer
	serverNoPort *MediaServer
	testStart    time.Time
}

func (s *ServerURLSuite) SetupTest() {
	s.SetupKeyComponents(s.T())
	s.SetupLoggerComponents()

	s.server = &MediaServer{Server: Server{
		hostname:   DefaultIP.String(),
		portManger: newPortManager(s.Logger, nil),
	}}
	err := s.server.SetPort(1337)
	s.Require().NoError(err)

	s.serverNoPort = &MediaServer{Server: Server{
		hostname:   DefaultIP.String(),
		portManger: newPortManager(s.Logger, nil),
	}}
	go func() {
		time.Sleep(waitTime)
		s.serverNoPort.port = 80
	}()

	s.testStart = time.Now()
}

// testNoPort takes two strings and compares expects them both to be equal
// then compares ServerURLSuite.testStart to the current time
// the difference must be greater than waitTime.
// This is caused by the ServerURLSuite.SetupTest waiting waitTime before unlocking the portWait sync.Mutex
func (s *ServerURLSuite) testNoPort(expected string, actual string) {
	s.Require().Equal(expected, actual)
	s.Require().Greater(time.Since(s.testStart), waitTime)
}

func (s *ServerURLSuite) TestServer_MakeBaseURL() {
	s.Require().Equal("https://127.0.0.1:1337", s.server.MakeBaseURL().String())
	s.testNoPort("https://127.0.0.1:80", s.serverNoPort.MakeBaseURL().String())
}

func (s *ServerURLSuite) TestServer_MakeImageServerURL() {
	s.Require().Equal("https://127.0.0.1:1337/messages/", s.server.MakeImageServerURL())
	s.testNoPort("https://127.0.0.1:80/messages/", s.serverNoPort.MakeImageServerURL())
}

func (s *ServerURLSuite) TestServer_MakeIdenticonURL() {
	s.Require().Equal(
		"https://127.0.0.1:1337/messages/identicons?publicKey=0xdaff0d11decade",
		s.server.MakeIdenticonURL("0xdaff0d11decade"))
	s.testNoPort(
		"https://127.0.0.1:80/messages/identicons?publicKey=0xdaff0d11decade",
		s.serverNoPort.MakeIdenticonURL("0xdaff0d11decade"))
}

func (s *ServerURLSuite) TestServer_MakeImageURL() {
	s.Require().Equal(
		"https://127.0.0.1:1337/messages/images?messageId=0x10aded70ffee",
		s.server.MakeImageURL("0x10aded70ffee"))

	s.testNoPort(
		"https://127.0.0.1:80/messages/images?messageId=0x10aded70ffee",
		s.serverNoPort.MakeImageURL("0x10aded70ffee"))
}

func (s *ServerURLSuite) TestServer_MakeAudioURL() {
	s.Require().Equal(
		"https://127.0.0.1:1337/messages/audio?messageId=0xde1e7ebee71e",
		s.server.MakeAudioURL("0xde1e7ebee71e"))
	s.testNoPort(
		"https://127.0.0.1:80/messages/audio?messageId=0xde1e7ebee71e",
		s.serverNoPort.MakeAudioURL("0xde1e7ebee71e"))
}

func (s *ServerURLSuite) TestServer_MakeStickerURL() {
	s.Require().Equal(
		"https://127.0.0.1:1337/ipfs?hash=0xdeadbeef4ac0",
		s.server.MakeStickerURL("0xdeadbeef4ac0"))
	s.testNoPort(
		"https://127.0.0.1:80/ipfs?hash=0xdeadbeef4ac0",
		s.serverNoPort.MakeStickerURL("0xdeadbeef4ac0"))
}
