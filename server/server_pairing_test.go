package server

import (
	"crypto/ecdsa"
	"crypto/rand"
	"regexp"
	"testing"

	"github.com/stretchr/testify/suite"
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
	err := s.PS.Start()
	s.Require().NoError(err)
	s.PS.ToBackground()
	s.PS.ToForeground()
	s.PS.ToBackground()
	s.PS.ToBackground()
	s.PS.ToForeground()
	s.PS.ToForeground()
	s.Require().Regexp(regexp.MustCompile("(https://\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}:\\d{1,5})"), s.PS.MakeBaseURL().String())
}

func (s *PairingServerSuite) TestPairingServer_StartPairing() {
	// Replace PairingServer.PayloadManager with a MockEncryptOnlyPayloadManager
	pm, err := NewMockEncryptOnlyPayloadManager(s.EphemeralAES)
	s.Require().NoError(err)
	s.PS.PayloadManager = pm

	modes := []Mode{
		Receiving,
		Sending,
	}

	for _, m := range modes {
		s.PS.mode = m

		err = s.PS.StartPairing()
		s.Require().NoError(err)

		cp, err := s.PS.MakeConnectionParams()
		s.Require().NoError(err)

		qr := cp.ToString()

		// Client reads QR code and parses the connection string
		ccp := new(ConnectionParams)
		err = ccp.FromString(qr)
		s.Require().NoError(err)

		c, err := NewPairingClient(ccp, nil)
		s.Require().NoError(err)

		// Compare cert values
		cert := c.serverCert
		cl := s.PS.cert.Leaf
		s.Require().Equal(cl.Signature, cert.Signature)
		s.Require().Zero(cl.PublicKey.(*ecdsa.PublicKey).X.Cmp(cert.PublicKey.(*ecdsa.PublicKey).X))
		s.Require().Zero(cl.PublicKey.(*ecdsa.PublicKey).Y.Cmp(cert.PublicKey.(*ecdsa.PublicKey).Y))
		s.Require().Equal(cl.Version, cert.Version)
		s.Require().Zero(cl.SerialNumber.Cmp(cert.SerialNumber))
		s.Require().Exactly(cl.NotBefore, cert.NotBefore)
		s.Require().Exactly(cl.NotAfter, cert.NotAfter)
		s.Require().Exactly(cl.IPAddresses, cert.IPAddresses)

		// Replace PairingClient.PayloadManager with a MockEncryptOnlyPayloadManager
		c.PayloadManager, err = NewMockEncryptOnlyPayloadManager(s.EphemeralAES)
		s.Require().NoError(err)

		err = c.PairAccount()
		s.Require().NoError(err)

		switch m {
		case Receiving:
			s.Require().Equal(c.PayloadManager.(*MockEncryptOnlyPayloadManager).toSend.plain, s.PS.Received())
			s.Require().Equal(s.PS.PayloadManager.(*MockEncryptOnlyPayloadManager).received.encrypted, c.PayloadManager.(*MockEncryptOnlyPayloadManager).toSend.encrypted)
			s.Require().Nil(s.PS.ToSend())
			s.Require().Nil(c.Received())
		case Sending:
			s.Require().Equal(c.Received(), s.PS.PayloadManager.(*MockEncryptOnlyPayloadManager).toSend.plain)
			s.Require().Equal(c.PayloadManager.(*MockEncryptOnlyPayloadManager).received.encrypted, s.PS.PayloadManager.(*MockEncryptOnlyPayloadManager).toSend.encrypted)
			s.Require().Nil(c.ToSend())
			s.Require().Nil(s.PS.Received())
		}

		// Reset the server's PayloadEncryptionManager
		s.PS.PayloadManager.(*MockEncryptOnlyPayloadManager).ResetPayload()
		s.PS.ResetPort()
	}
}

func (s *PairingServerSuite) sendingSetup() *PairingClient {
	// Replace PairingServer.PayloadManager with a MockEncryptOnlyPayloadManager
	pm, err := NewMockEncryptOnlyPayloadManager(s.EphemeralAES)
	s.Require().NoError(err)
	s.PS.PayloadManager = pm
	s.PS.mode = Sending

	err = s.PS.StartPairing()
	s.Require().NoError(err)

	cp, err := s.PS.MakeConnectionParams()
	s.Require().NoError(err)

	qr := cp.ToString()

	// Client reads QR code and parses the connection string
	ccp := new(ConnectionParams)
	err = ccp.FromString(qr)
	s.Require().NoError(err)

	c, err := NewPairingClient(ccp, nil)
	s.Require().NoError(err)

	// Replace PairingClient.PayloadManager with a MockEncryptOnlyPayloadManager
	c.PayloadManager, err = NewMockEncryptOnlyPayloadManager(s.EphemeralAES)
	s.Require().NoError(err)

	return c
}

func (s *PairingServerSuite) TestPairingServer_handlePairingChallengeMiddleware() {
	c := s.sendingSetup()

	// Attempt to get the private key data, this should fail because there is no challenge
	err := c.receiveAccountData()
	s.Require().Error(err)
	s.Require().Equal("status not ok, received '403 Forbidden'", err.Error())

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
	s.Require().Equal("status not ok, received '403 Forbidden'", err.Error())

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
	s.Require().Equal("status not ok, received '403 Forbidden'", err.Error())

	// Get the real challenge
	err = c.getChallenge()
	s.Require().NoError(err)

	// Attempt to get the account data, should fail because the client is now blocked.
	err = c.receiveAccountData()
	s.Require().Error(err)
	s.Require().Equal("status not ok, received '403 Forbidden'", err.Error())
}
