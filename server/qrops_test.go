package server

import (
	"encoding/base64"
	"io/ioutil"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/server/servertest"
)

type QROpsTestSuite struct {
	suite.Suite
	servertest.TestKeyComponents
	servertest.TestLoggerComponents

	server          *MediaServer
	serverNoPort    *MediaServer
	testStart       time.Time
	multiaccountsDB *multiaccounts.Database
}

var (
	keyUID = "0xdeadbeef"
	qrURL  = "https://github.com/status-im/status-go/pull/3154"
)

func TestQROpsTestSuite(t *testing.T) {
	suite.Run(t, new(QROpsTestSuite))
}

func (s *QROpsTestSuite) SetupTest() {
	s.SetupKeyComponents(s.T())
	s.SetupLoggerComponents()

	mediaServer, err := NewMediaServer(nil, nil, nil)
	s.Require().NoError(err)

	s.server = mediaServer
	err = s.server.SetPort(customPortForTests)
	s.Require().NoError(err)

	err = s.server.Start()
	s.Require().NoError(err)

	s.serverNoPort = &MediaServer{Server: Server{
		hostname:   LocalHostIP.String(),
		portManger: newPortManager(s.Logger, nil),
	}}
	go func() {
		time.Sleep(waitTime)
		s.serverNoPort.port = 80
	}()

	s.testStart = time.Now()
}

func seedTestDBWithIdentityImagesForQRCodeTests(s *QROpsTestSuite, db *multiaccounts.Database, keyUID string) {
	iis := images.SampleIdentityImageForQRCode()
	err := db.StoreIdentityImages(keyUID, iis, false)
	s.Require().NoError(err)
}

// TestQROpsCodeWithoutSuperImposingLogo tests the QR code generation logic, it also allows us to debug in case
// things go wrong, here we compare generate a new QR code and compare its bytes with an already generated one.
func (s *QROpsTestSuite) TestQROpsCodeWithoutSuperImposingLogo() {

	var params = url.Values{}
	params.Set("url", base64.StdEncoding.EncodeToString([]byte(qrURL)))
	params.Set("allowProfileImage", "false")
	params.Set("level", "2")
	params.Set("size", "200")
	params.Set("imageName", "")

	payload := generateQRBytes(params, s.Logger, s.multiaccountsDB)
	expectedPayload, err := images.Asset("_assets/tests/qr/defaultQR.png")

	s.Require().NoError(err)
	s.Require().NotEmpty(payload)
	require.Equal(s.T(), payload, expectedPayload)
}

func (s *QROpsTestSuite) TestQROpsCodeWithSuperImposingLogo() {

	tmpfile, err := ioutil.TempFile("", "accounts-tests-")
	s.Require().NoError(err)

	db, err := multiaccounts.InitializeDB(tmpfile.Name())
	s.Require().NoError(err)

	seedTestDBWithIdentityImagesForQRCodeTests(s, db, keyUID)

	var params = url.Values{}
	params.Set("url", base64.StdEncoding.EncodeToString([]byte(qrURL)))
	params.Set("allowProfileImage", "true")
	params.Set("level", "2")
	params.Set("size", "200")
	params.Set("keyUid", keyUID)
	params.Set("imageName", "large")

	payload := generateQRBytes(params, s.Logger, db)
	s.Require().NotEmpty(payload)

	expectedPayload, err := images.Asset("_assets/tests/qr/QRWithLogo.png")
	require.Equal(s.T(), payload, expectedPayload)
	s.Require().NoError(err)

	err = db.Close()
	s.Require().NoError(err)

	err = os.Remove(tmpfile.Name())
	s.Require().NoError(err)
}
