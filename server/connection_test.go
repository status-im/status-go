package server

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/suite"
)

var (
	connectionString = "2:4FHRnp:Q4:6jpbvo2ucrtrnpXXF4DQYuysh697isH9ppd2aT8uSRDh:eQUriVtGtkWhPJFeLZjF:3"
)

func TestConnectionParamsSuite(t *testing.T) {
	suite.Run(t, new(ConnectionParamsSuite))
}

type ConnectionParamsSuite struct {
	suite.Suite
	TestKeyComponents
	TestCertComponents

	server *PairingServer
}

func (s *ConnectionParamsSuite) SetupSuite() {
	s.SetupKeyComponents(s.T())
	s.SetupCertComponents(s.T())

	cert, _, err := GenerateCertFromKey(s.PK, s.NotBefore, defaultIP.String())
	s.Require().NoError(err)

	bs := NewServer(&cert, defaultIP.String())
	bs.port = 1337

	s.server = &PairingServer{
		Server: bs,
		pk:     s.PK,
		mode:   Sending,
	}
}

func (s *ConnectionParamsSuite) TestConnectionParams_ToString() {
	cp, err := s.server.MakeConnectionParams()
	s.Require().NoError(err)

	cps, err := cp.ToString()
	s.Require().NoError(err)

	s.Require().Equal(connectionString, cps)
}

func (s *ConnectionParamsSuite) TestConnectionParams_Generate() {
	cp := new(ConnectionParams)
	err := cp.FromString(connectionString)
	s.Require().NoError(err)

	s.Require().Exactly(Sending, cp.serverMode)

	u, c, err := cp.Generate()
	s.Require().NoError(err)

	s.Require().Equal("https://127.0.0.1:1337", u.String())
	s.Require().Equal(defaultIP.String(), u.Hostname())
	s.Require().Equal("1337", u.Port())

	// Parse cert PEM into x509 cert
	block, _ := pem.Decode(c)
	s.Require().NotNil(block)
	cert, err := x509.ParseCertificate(block.Bytes)
	s.Require().NoError(err)

	// Compare cert values
	cl := s.server.cert.Leaf
	s.Require().NotEqual(cl.Signature, cert.Signature)
	s.Require().Zero(cl.PublicKey.(*ecdsa.PublicKey).X.Cmp(cert.PublicKey.(*ecdsa.PublicKey).X))
	s.Require().Zero(cl.PublicKey.(*ecdsa.PublicKey).Y.Cmp(cert.PublicKey.(*ecdsa.PublicKey).Y))
	s.Require().Equal(cl.Version, cert.Version)
	s.Require().Zero(cl.SerialNumber.Cmp(cert.SerialNumber))
	s.Require().Exactly(cl.NotBefore, cert.NotBefore)
	s.Require().Exactly(cl.NotAfter, cert.NotAfter)
	s.Require().Exactly(cl.IPAddresses, cert.IPAddresses)
}
