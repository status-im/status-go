package server

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

var (
	connectionString = "2:4FHRnp:Q4:uqnnMwVUfJc2Fkcaojet8F1ufKC3hZdGEt47joyBx9yd:BbnZ7Gc66t54a9kEFCf7FW8SGQuYypwHVeNkRYeNoqV6:3"
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
		pk:     &s.PK.PublicKey,
		ek:     s.AES,
		mode:   Sending,
	}
}

func (s *ConnectionParamsSuite) TestConnectionParams_ToString() {
	cp, err := s.server.MakeConnectionParams()
	s.Require().NoError(err)

	cps := cp.ToString()
	s.Require().Equal(connectionString, cps)
}

func (s *ConnectionParamsSuite) TestConnectionParams_Generate() {
	cp := new(ConnectionParams)
	err := cp.FromString(connectionString)
	s.Require().NoError(err)

	s.Require().Exactly(Sending, cp.serverMode)

	u, err := cp.URL()
	s.Require().NoError(err)

	s.Require().Equal("https://127.0.0.1:1337", u.String())
	s.Require().Equal(defaultIP.String(), u.Hostname())
	s.Require().Equal("1337", u.Port())

	s.Require().True(cp.publicKey.Equal(&s.PK.PublicKey))
	s.Require().Equal(s.AES, cp.aesKey)
}
