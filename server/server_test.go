package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
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
		hostname:   defaultIP.String(),
		portManger: newPortManager(s.Logger, nil),
	}}
	err := s.server.SetPort(1337)
	s.Require().NoError(err)

	s.serverNoPort = &MediaServer{Server: Server{
		hostname:   defaultIP.String(),
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
	s.Require().Greater(time.Now().Sub(s.testStart), waitTime)
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

func (s *ServerURLSuite) TestServer_MakeQRURL(t *testing.T) {
	// todo : bootstrap with keyuid which contains profile picture
	jsonBody := []byte(``)
	bodyReader := bytes.NewReader(jsonBody)

	db, stop := setupTestDB(t)
	defer stop()
	multiaccounts/database_test.go:74
	database_test.seedTestDBWithIdentityImages(t, db, keyUID)

	requestURL := fmt.Sprintf("https://127.0.0.1:1337/QRImages?qrurl=aHR0cHM6Ly9naXRodWIuY29tL3llcW93bi9nby1xcmNvZGUv&keyUid=&imageName=thumbnail")
	response, err := http.NewRequest(http.MethodGet, requestURL, bodyReader)
	payload, err := ioutil.ReadAll(response.Body)
	s.Require().NotEmpty(response)
	s.Require().NotEmpty(payload)
	s.Require().NoError(err)

	//requestURLTwo := fmt.Sprintf("https://localhost:1337/QRImagesWithLogo?qrurl=Y3MyOjV2ZDZTTDpLRkM6MjZnQW91VTZENkE0ZENzOUxLN2pIbVhaM2dqVmRQY3p2WDd5ZXVzWlJIVGVSOjNIeEo5UXI0SDM1MWRQb1hqUVlzZFBYNHRLNnRWNlRrZHNIazF4TVpFWm1MOjM=&keyUid=0xe98e17415acf3fa4145667bfc8fd259ae780fc58b6f722b4a21c604c71a68407&imageName=thumbnail")
	//response, err := http.NewRequest(http.MethodGet, requestURLTwo, bodyReader)

	s.Require().NotEmpty(response)
	s.Require().NoError(err)

	s.Require().Equal(
		"https://127.0.0.1:1337/QRImages?qurul=cs2%3A5vd6SL%3AKFC%3A26gAouU6D6A4dCs9LK7jHmXZ3gjVdPczvX7yeusZRHTeR%3A3HxJ9Qr4H351dPoXjQYsdPX4tK6tV6TkdsHk1xMZEZmL%3A3",
		s.server.MakeQRURL("cs2:5vd6SL:KFC:26gAouU6D6A4dCs9LK7jHmXZ3gjVdPczvX7yeusZRHTeR:3HxJ9Qr4H351dPoXjQYsdPX4tK6tV6TkdsHk1xMZEZmL:3"))
	s.Require().Equal(
		"https://127.0.0.1:80/QRImages?qurul=cs2%3A5vd6SL%3AKFC%3A26gAouU6D6A4dCs9LK7jHmXZ3gjVdPczvX7yeusZRHTeR%3A3HxJ9Qr4H351dPoXjQYsdPX4tK6tV6TkdsHk1xMZEZmL%3A3",
		s.serverNoPort.MakeQRURL("cs2:5vd6SL:KFC:26gAouU6D6A4dCs9LK7jHmXZ3gjVdPczvX7yeusZRHTeR:3HxJ9Qr4H351dPoXjQYsdPX4tK6tV6TkdsHk1xMZEZmL:3"))
}
