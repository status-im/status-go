package pairing

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/server"
)

func TestPairingServerSuite(t *testing.T) {
	suite.Run(t, new(PairingServerSuite))
}

type PairingServerSuite struct {
	suite.Suite
	TestPairingServerComponents
}

func (s *PairingServerSuite) SetupTest() {
	s.SetupPairingServerComponents(s.T())
}

func (s *PairingServerSuite) TestMultiBackgroundForeground() {
	err := s.SS.Start()
	s.Require().NoError(err)
	s.SS.ToBackground()
	s.SS.ToForeground()
	s.SS.ToBackground()
	s.SS.ToBackground()
	s.SS.ToForeground()
	s.SS.ToForeground()
	s.Require().Regexp(regexp.MustCompile("(https://\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}:\\d{1,5})"), s.SS.MakeBaseURL().String()) // nolint: gosimple
}

func (s *PairingServerSuite) TestMultiTimeout() {
	s.SS.SetTimeout(20)

	err := s.SS.Start()
	s.Require().NoError(err)

	s.SS.ToBackground()
	s.SS.ToForeground()
	s.SS.ToBackground()
	s.SS.ToBackground()
	s.SS.ToForeground()
	s.SS.ToForeground()

	s.Require().Regexp(regexp.MustCompile("(https://\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}:\\d{1,5})"), s.SS.MakeBaseURL().String()) // nolint: gosimple

	time.Sleep(7 * time.Millisecond)
	s.SS.ToBackground()
	time.Sleep(7 * time.Millisecond)
	s.SS.ToForeground()
	time.Sleep(7 * time.Millisecond)
	s.SS.ToBackground()
	time.Sleep(7 * time.Millisecond)
	s.SS.ToBackground()
	time.Sleep(7 * time.Millisecond)
	s.SS.ToForeground()
	time.Sleep(7 * time.Millisecond)
	s.SS.ToForeground()

	// Wait for timeout to expire
	time.Sleep(40 * time.Millisecond)
	s.Require().False(s.SS.IsRunning())
}

// TestPairingServer_StartPairingSend tests that a Server can send data to a ReceiverClient
func (s *PairingServerSuite) TestPairingServer_StartPairingSend() {
	// Replace PairingServer.accountMounter with a MockPayloadMounter
	pm := NewMockPayloadMounter(s.EphemeralAES)
	s.SS.accountMounter = pm
	s.SS.mode = Sending

	err := s.SS.startSendingData()
	s.Require().NoError(err)

	cp, err := s.SS.MakeConnectionParams()
	s.Require().NoError(err)

	qr := cp.ToString()

	// Client reads QR code and parses the connection string
	ccp := new(ConnectionParams)
	err = ccp.FromString(qr)
	s.Require().NoError(err)

	c, err := NewReceiverClient(nil, ccp, NewReceiverClientConfig())
	s.Require().NoError(err)

	// Compare cert values
	cert := c.serverCert
	cl := s.SS.GetCert().Leaf
	s.Require().Equal(cl.Signature, cert.Signature)
	s.Require().Zero(cl.PublicKey.(*ecdsa.PublicKey).X.Cmp(cert.PublicKey.(*ecdsa.PublicKey).X))
	s.Require().Zero(cl.PublicKey.(*ecdsa.PublicKey).Y.Cmp(cert.PublicKey.(*ecdsa.PublicKey).Y))
	s.Require().Equal(cl.Version, cert.Version)
	s.Require().Zero(cl.SerialNumber.Cmp(cert.SerialNumber))
	s.Require().Exactly(cl.NotBefore, cert.NotBefore)
	s.Require().Exactly(cl.NotAfter, cert.NotAfter)
	s.Require().Exactly(cl.IPAddresses, cert.IPAddresses)

	// Replace ReceivingClient.accountReceiver with a MockPayloadReceiver
	c.accountReceiver = NewMockPayloadReceiver(s.EphemeralAES)

	err = c.getChallenge()
	s.Require().NoError(err)
	err = c.receiveAccountData()
	s.Require().NoError(err)

	s.Require().Equal(c.accountReceiver.Received(), s.SS.accountMounter.(*MockPayloadMounter).encryptor.payload.plain)
	s.Require().Equal(c.accountReceiver.(*MockPayloadReceiver).encryptor.payload.encrypted, s.SS.accountMounter.(*MockPayloadMounter).encryptor.payload.encrypted)
}

// TestPairingServer_StartPairingReceive tests that a Server can receive data to a SenderClient
func (s *PairingServerSuite) TestPairingServer_StartPairingReceive() {
	// Replace PairingServer.PayloadManager with a MockEncryptOnlyPayloadManager
	pm := NewMockPayloadReceiver(s.EphemeralAES)
	s.RS.accountReceiver = pm

	s.RS.mode = Receiving

	err := s.RS.startReceivingData()
	s.Require().NoError(err)

	cp, err := s.RS.MakeConnectionParams()
	s.Require().NoError(err)

	qr := cp.ToString()

	// Client reads QR code and parses the connection string
	ccp := new(ConnectionParams)
	err = ccp.FromString(qr)
	s.Require().NoError(err)

	c, err := NewSenderClient(nil, ccp, &SenderClientConfig{Sender: &SenderConfig{}, Client: &ClientConfig{}})
	s.Require().NoError(err)

	// Compare cert values
	cert := c.serverCert
	cl := s.RS.GetCert().Leaf
	s.Require().Equal(cl.Signature, cert.Signature)
	s.Require().Zero(cl.PublicKey.(*ecdsa.PublicKey).X.Cmp(cert.PublicKey.(*ecdsa.PublicKey).X))
	s.Require().Zero(cl.PublicKey.(*ecdsa.PublicKey).Y.Cmp(cert.PublicKey.(*ecdsa.PublicKey).Y))
	s.Require().Equal(cl.Version, cert.Version)
	s.Require().Zero(cl.SerialNumber.Cmp(cert.SerialNumber))
	s.Require().Exactly(cl.NotBefore, cert.NotBefore)
	s.Require().Exactly(cl.NotAfter, cert.NotAfter)
	s.Require().Exactly(cl.IPAddresses, cert.IPAddresses)

	// Replace SendingClient.accountMounter with a MockPayloadMounter
	c.accountMounter = NewMockPayloadMounter(s.EphemeralAES)
	s.Require().NoError(err)

	err = c.sendAccountData()
	s.Require().NoError(err)

	s.Require().Equal(c.accountMounter.(*MockPayloadMounter).encryptor.payload.plain, s.RS.accountReceiver.Received())
	s.Require().Equal(s.RS.accountReceiver.(*MockPayloadReceiver).encryptor.getEncrypted(), c.accountMounter.(*MockPayloadMounter).encryptor.payload.encrypted)
}

func (s *PairingServerSuite) sendingSetup() *ReceiverClient {
	// Replace PairingServer.PayloadManager with a MockPayloadReceiver
	pm := NewMockPayloadMounter(s.EphemeralAES)
	s.SS.accountMounter = pm
	s.SS.mode = Sending

	err := s.SS.startSendingData()
	s.Require().NoError(err)

	cp, err := s.SS.MakeConnectionParams()
	s.Require().NoError(err)

	qr := cp.ToString()

	// Client reads QR code and parses the connection string
	ccp := new(ConnectionParams)
	err = ccp.FromString(qr)
	s.Require().NoError(err)

	c, err := NewReceiverClient(nil, ccp, NewReceiverClientConfig())
	s.Require().NoError(err)

	// Replace PairingClient.PayloadManager with a MockEncryptOnlyPayloadManager
	c.accountReceiver = NewMockPayloadReceiver(s.EphemeralAES)
	s.Require().NoError(err)

	return c
}

func (s *PairingServerSuite) TestPairingServer_handlePairingChallengeMiddleware() {
	c := s.sendingSetup()

	// Attempt to get the private key data, this should fail because there is no challenge
	err := c.receiveAccountData()
	s.Require().Error(err)
	s.Require().Equal("[client] status not ok when receiving account data, received '403 Forbidden'", err.Error())

	err = c.getChallenge()
	s.Require().NoError(err)
	challenge := c.serverChallenge

	// This is NOT a mistake! Call c.getChallenge() twice to check that the client gets the same challenge
	// the server will only generate 1 challenge per session per connection
	err = c.getChallenge()
	s.Require().NoError(err)
	s.Require().Equal(challenge, c.serverChallenge)

	// receiving account data should now work.
	err = c.receiveAccountData()
	s.Require().NoError(err)
}

func (s *PairingServerSuite) TestPairingServer_handlePairingChallengeMiddleware_block() {
	c := s.sendingSetup()

	// Attempt to get the private key data, this should fail because there is no challenge
	err := c.receiveAccountData()
	s.Require().Error(err)
	s.Require().Equal("[client] status not ok when receiving account data, received '403 Forbidden'", err.Error())

	// Get the challenge
	err = c.getChallenge()
	s.Require().NoError(err)

	// Simulate encrypting with a dodgy key, write some nonsense to the challenge field
	c.serverChallenge = make([]byte, 64)
	_, err = rand.Read(c.serverChallenge)
	s.Require().NoError(err)

	// Attempt again to get the account data, should fail
	// behind the scenes the server will block the session if the client fails the challenge. There is no forgiveness!
	err = c.receiveAccountData()
	s.Require().Error(err)
	s.Require().Equal("[client] status not ok when receiving account data, received '403 Forbidden'", err.Error())

	// Get the real challenge
	err = c.getChallenge()
	s.Require().NoError(err)

	// Attempt to get the account data, should fail because the client is now blocked.
	err = c.receiveAccountData()
	s.Require().Error(err)
	s.Require().Equal("[client] status not ok when receiving account data, received '403 Forbidden'", err.Error())
}

func testHandler(t *testing.T) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		say, ok := r.URL.Query()["say"]
		if !ok || len(say) == 0 {
			say = append(say, "nothing")
		}

		_, err := w.Write([]byte("Hello I like to be a tls server. You said: `" + say[0] + "` " + time.Now().String()))
		if err != nil {
			require.NoError(t, err)
		}
	}
}

func makeThingToSay() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func (s *PairingServerSuite) TestGetOutboundIPWithFullServerE2e() {
	s.SS.mode = Sending
	s.SS.SetHandlers(server.HandlerPatternMap{"/hello": testHandler(s.T())})

	err := s.SS.Start()
	s.Require().NoError(err)

	// Give time for the sever to be ready, hacky I know, I'll iron this out
	time.Sleep(100 * time.Millisecond)

	// Server generates a QR code connection string
	cp, err := s.SS.MakeConnectionParams()
	s.Require().NoError(err)

	qr := cp.ToString()

	// Client reads QR code and parses the connection string
	ccp := new(ConnectionParams)
	err = ccp.FromString(qr)
	s.Require().NoError(err)

	c, err := NewReceiverClient(nil, ccp, NewReceiverClientConfig())
	s.Require().NoError(err)

	thing, err := makeThingToSay()
	s.Require().NoError(err)

	response, err := c.Get(c.baseAddress.String() + "/hello?say=" + thing)
	s.Require().NoError(err)

	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	s.Require().NoError(err)
	s.Require().Equal("Hello I like to be a tls server. You said: `"+thing+"`", string(content[:109]))
}
