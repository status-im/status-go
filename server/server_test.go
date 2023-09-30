package server

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/images"
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

	mediaServer, err := NewMediaServer(nil, nil, nil)
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
		baseURLWithCustomPort+"/status-link-preview/thumbnail?message-id=99&url=https%3A%2F%2Fstatus.app",
		s.server.MakeStatusLinkPreviewThumbnailURL("99", "https://status.app", "contact-icon"))

	s.testNoPort(
		baseURLWithDefaultPort+"/status-link-preview/thumbnail?message-id=99&url=https%3A%2F%2Fstatus.app",
		s.serverNoPort.MakeStatusLinkPreviewThumbnailURL("99", "https://status.app", "contact-icon"))
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

// TestQRCodeGeneration tests if we provide all the correct parameters to the media server
// do we get a valid QR code or not as part of the response payload.
// we have stored a generated QR code in tests folder, and we compare their bytes.
func (s *ServerURLSuite) TestQRCodeGeneration() {

	qrURL := "https://github.com/status-im/status-go/pull/3154"
	generatedURL := base64.StdEncoding.EncodeToString([]byte(qrURL))
	generatedURL = s.serverForQR.MakeQRURL(generatedURL, "false", "2", "200", "", "")

	u, err := url.Parse(generatedURL)
	if err != nil {
		s.Require().NoError(err)
	}

	if u.Scheme == "" || u.Host == "" {
		s.Require().Failf("generatedURL is not a valid URL: %s", generatedURL)
	}

	serverCert := s.serverForQR.cert
	serverCertBytes := serverCert.Certificate[0]

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertBytes})

	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		s.Require().NoError(err)
	}

	_ = rootCAs.AppendCertsFromPEM(certPem)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    rootCAs,
		},
	}

	client := &http.Client{Transport: tr}

	req, err := http.NewRequest(http.MethodGet, generatedURL, nil)
	if err != nil {
		s.Require().NoError(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		s.Require().NoError(err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		s.Require().Failf("Unexpected response status code: %d", fmt.Sprint(resp.StatusCode))
	}

	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.Require().NoError(err)
	}

	s.Require().NotEmpty(payload)

	expectedPayload, err := images.Asset("_assets/tests/qr/defaultQR.png")
	require.Equal(s.T(), payload, expectedPayload)
	s.Require().NoError(err)

	//(siddarthkay) un-comment code block below to generate the file in tests folder
	//f, err := os.Create("image.png")
	//if err != nil {
	//	s.Require().NoError(err)
	//
	//}
	//defer f.Close()
	//_, err = f.Write(payload)
	//
	//if err != nil {
	//	s.Require().NoError(err)
	//}
}
