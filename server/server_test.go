package server

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/server/servertest"
)

const (
	waitTime            = 50 * time.Millisecond
	customPortForTests  = 1337
	defaultPortForTests = 80
)

var (
	baseURL                = "https://127.0.0.1"
	baseURLWithCustomPort  = fmt.Sprintf("%s:%d", baseURL, customPortForTests)
	baseURLWithDefaultPort = fmt.Sprintf("%s:%d", baseURL, defaultPortForTests)
)

func TestServerURLSuite(t *testing.T) {
	suite.Run(t, new(ServerURLSuite))
}

type ServerURLSuite struct {
	suite.Suite
	servertest.TestKeyComponents
	servertest.TestLoggerComponents

	server       *MediaServer
	serverForQR  *MediaServer
	serverNoPort *MediaServer
	testStart    time.Time
}

func (s *ServerURLSuite) SetupTest() {
	s.SetupKeyComponents(s.T())
	s.SetupLoggerComponents()

	mediaServer, err := NewMediaServer(nil, nil, nil, nil)
	s.Require().NoError(err)

	s.serverForQR = mediaServer

	err = s.serverForQR.Start()
	s.Require().NoError(err)

	s.server = &MediaServer{Server: Server{
		hostname:   LocalHostIP.String(),
		portManger: newPortManager(s.Logger, nil),
	}}
	err = s.server.SetPort(customPortForTests)
	s.Require().NoError(err)

	s.serverNoPort = &MediaServer{Server: Server{
		hostname:   LocalHostIP.String(),
		portManger: newPortManager(s.Logger, nil),
	}}
	go func() {
		time.Sleep(waitTime)
		s.serverNoPort.port = defaultPortForTests
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
	s.Require().Equal(baseURLWithCustomPort, s.server.MakeBaseURL().String())
	s.testNoPort(baseURLWithDefaultPort, s.serverNoPort.MakeBaseURL().String())
}

func (s *ServerURLSuite) TestServer_MakeImageServerURL() {
	s.Require().Equal(baseURLWithCustomPort+"/messages/", s.server.MakeImageServerURL())
	s.testNoPort(baseURLWithDefaultPort+"/messages/", s.serverNoPort.MakeImageServerURL())
}

func (s *ServerURLSuite) TestServer_MakeImageURL() {
	s.Require().Equal(
		baseURLWithCustomPort+"/messages/images?messageId=0x10aded70ffee",
		s.server.MakeImageURL("0x10aded70ffee"))

	s.testNoPort(
		baseURLWithDefaultPort+"/messages/images?messageId=0x10aded70ffee",
		s.serverNoPort.MakeImageURL("0x10aded70ffee"))
}

func (s *ServerURLSuite) TestServer_MakeLinkPreviewThumbnailURL() {
	s.Require().Equal(
		baseURLWithCustomPort+"/link-preview/thumbnail?message-id=99&url=https%3A%2F%2Fgithub.com",
		s.server.MakeLinkPreviewThumbnailURL("99", "https://github.com"))

	s.testNoPort(
		baseURLWithDefaultPort+"/link-preview/thumbnail?message-id=99&url=https%3A%2F%2Fgithub.com",
		s.serverNoPort.MakeLinkPreviewThumbnailURL("99", "https://github.com"))
}

func (s *ServerURLSuite) TestServer_MakeStatusLinkPreviewThumbnailURL() {
	s.Require().Equal(
		baseURLWithCustomPort+"/status-link-preview/thumbnail?image-id=contact-icon&message-id=99&url=https%3A%2F%2Fstatus.app",
		s.server.MakeStatusLinkPreviewThumbnailURL("99", "https://status.app", common.MediaServerContactIcon))

	s.testNoPort(
		baseURLWithDefaultPort+"/status-link-preview/thumbnail?image-id=contact-icon&message-id=99&url=https%3A%2F%2Fstatus.app",
		s.serverNoPort.MakeStatusLinkPreviewThumbnailURL("99", "https://status.app", common.MediaServerContactIcon))
}

func (s *ServerURLSuite) TestServer_MakeAudioURL() {
	s.Require().Equal(
		baseURLWithCustomPort+"/messages/audio?messageId=0xde1e7ebee71e",
		s.server.MakeAudioURL("0xde1e7ebee71e"))
	s.testNoPort(
		baseURLWithDefaultPort+"/messages/audio?messageId=0xde1e7ebee71e",
		s.serverNoPort.MakeAudioURL("0xde1e7ebee71e"))
}

func (s *ServerURLSuite) TestServer_MakeStickerURL() {
	s.Require().Equal(
		baseURLWithCustomPort+"/ipfs?hash=0xdeadbeef4ac0",
		s.server.MakeStickerURL("0xdeadbeef4ac0"))
	s.testNoPort(
		baseURLWithDefaultPort+"/ipfs?hash=0xdeadbeef4ac0",
		s.serverNoPort.MakeStickerURL("0xdeadbeef4ac0"))
}

func (s *ServerURLSuite) TestServer_MakeContactImageURL() {
	s.Require().Equal(
		baseURLWithCustomPort+"/contactImages?clock=1&imageName=Test&publicKey=0x1",
		s.server.MakeContactImageURL("0x1", "Test", uint64(1)))
	s.testNoPort(
		baseURLWithDefaultPort+"/contactImages?clock=1&imageName=Test&publicKey=0x1",
		s.serverNoPort.MakeContactImageURL("0x1", "Test", uint64(1)))
}
