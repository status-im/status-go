package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

func TestPairingServerSuite(t *testing.T) {
	suite.Run(t, new(PairingServerSuite))
}

type PairingServerSuite struct {
	suite.Suite
	TestPairingServerComponents
}

func (s *PairingServerSuite) SetupSuite() {
	s.SetupPairingServerComponents(s.T())
}

func (s *PairingServerSuite) TestPairingServer_StartPairing() {
	// Replace PairingServer.PayloadManager with a MockEncryptOnlyPayloadManager
	pm, err := NewMockEncryptOnlyPayloadManager(s.EphemeralPK)
	s.Require().NoError(err)
	s.PS.PayloadManager = pm

	modes := []Mode{
		Receiving,
		Sending,
	}

	for _, m := range modes {
		s.PS.mode = m

		if m == Sending {
			err := s.PS.Mount()
			s.Require().NoError(err)
		}

		err = s.PS.StartPairing()
		s.Require().NoError(err)

		// Give time for the sever to be ready, hacky I know, I'll iron this out
		time.Sleep(10 * time.Millisecond)

		cp, err := s.PS.MakeConnectionParams()
		s.Require().NoError(err)

		qr, err := cp.ToString()
		s.Require().NoError(err)

		// Client reads QR code and parses the connection string
		ccp := new(ConnectionParams)
		err = ccp.FromString(qr)
		s.Require().NoError(err)

		c, err := NewPairingClient(ccp, nil)
		s.Require().NoError(err)

		// Replace PairingClient.PayloadManager with a MockEncryptOnlyPayloadManager
		c.PayloadManager, err = NewMockEncryptOnlyPayloadManager(s.EphemeralPK)
		s.Require().NoError(err)

		if m == Receiving {
			err := c.Mount()
			s.Require().NoError(err)
		}

		err = c.PairAccount()
		s.Require().NoError(err)

		switch m {
		case Receiving:
			s.Require().Equal(c.PayloadManager.(*MockEncryptOnlyPayloadManager).pem.toSend.plain, s.PS.Received())
			s.Require().Equal(s.PS.PayloadManager.(*MockEncryptOnlyPayloadManager).pem.received.encrypted, c.PayloadManager.(*MockEncryptOnlyPayloadManager).pem.toSend.encrypted)
			s.Require().Nil(s.PS.ToSend())
			s.Require().Nil(c.Received())
		case Sending:
			s.Require().Equal(c.Received(), s.PS.PayloadManager.(*MockEncryptOnlyPayloadManager).pem.toSend.plain)
			s.Require().Equal(c.PayloadManager.(*MockEncryptOnlyPayloadManager).pem.received.encrypted, s.PS.PayloadManager.(*MockEncryptOnlyPayloadManager).pem.toSend.encrypted)
			s.Require().Nil(c.ToSend())
			s.Require().Nil(s.PS.Received())
		}

		// Reset the server's PayloadEncryptionManager
		s.PS.PayloadManager.(*MockEncryptOnlyPayloadManager).ResetPayload()
	}
}
