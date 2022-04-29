package server

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/suite"
)

func TestServerURLSuite(t *testing.T) {
	suite.Run(t, new(ServerURLSuite))
}

type ServerURLSuite struct {
	suite.Suite
	TestKeyComponents
	TestCertComponents

	server       *MediaServer
	serverNoPort *MediaServer
	pairingServer    *PairingServer
}

func (s *ServerURLSuite) SetupSuite() {
	s.SetupKeyComponents(s.T())
	s.SetupCertComponents(s.T())

	cert, _, err := GenerateCertFromKey(s.PK, s.NotBefore, defaultIP.String())
	s.Require().NoError(err)

	s.server = &MediaServer{Server: Server{
		hostname: defaultIP.String(),
		port:     1337,
	}}
	s.serverNoPort = &MediaServer{Server: Server{
		hostname: defaultIP.String(),
	}}
	s.pairingServer = &PairingServer{
		Server: Server{cert: &cert, hostname: defaultIP.String()},
		pk:       s.PK,
	}
}

func (s *ServerURLSuite) TestServer_MakeQRData() {
	qr, err := s.pairingServer.MakeQRData()
	s.Require().NoError(err)

	s.Require().Regexp(
		"^4FHRnp:[1-9|A-Z|a-z]{1,4}:6jpbvo2ucrtrnpXXF4DQYuysh697isH9ppd2aT8uSRDh:eQUriVtGtkWhPJFeLZjF$",
		qr)
}

func (s *ServerURLSuite) TestServer_ParseQRData() {
	u, c, err := ParseQRData("4FHRnp:H6G:6jpbvo2ucrtrnpXXF4DQYuysh697isH9ppd2aT8uSRDh:eQUriVtGtkWhPJFeLZjF")
	s.Require().NoError(err)

	s.Require().Equal("https://127.0.0.1:54129", u.String())
	s.Require().Equal(defaultIP.String(), u.Hostname())
	s.Require().Equal("54129", u.Port())

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

	spew.Dump(cl, cert)
}

func (s *ServerURLSuite) TestServer_MakeBaseURL() {
	s.Require().Equal("https://127.0.0.1:1337", s.server.MakeBaseURL().String())
	s.Require().Equal("https://127.0.0.1:0", s.serverNoPort.MakeBaseURL().String())
}

func (s *ServerURLSuite) TestServer_MakeImageServerURL() {
	s.Require().Equal("https://127.0.0.1:1337/messages/", s.server.MakeImageServerURL())
	s.Require().Equal("https://127.0.0.1:0/messages/", s.serverNoPort.MakeImageServerURL())
}

func (s *ServerURLSuite) TestServer_MakeIdenticonURL() {
	s.Require().Equal(
		"https://127.0.0.1:1337/messages/identicons?publicKey=0xdaff0d11decade",
		s.server.MakeIdenticonURL("0xdaff0d11decade"))
	s.Require().Equal(
		"https://127.0.0.1:0/messages/identicons?publicKey=0xdaff0d11decade",
		s.serverNoPort.MakeIdenticonURL("0xdaff0d11decade"))
}

func (s *ServerURLSuite) TestServer_MakeImageURL() {
	s.Require().Equal(
		"https://127.0.0.1:1337/messages/images?messageId=0x10aded70ffee",
		s.server.MakeImageURL("0x10aded70ffee"))
	s.Require().Equal(
		"https://127.0.0.1:0/messages/images?messageId=0x10aded70ffee",
		s.serverNoPort.MakeImageURL("0x10aded70ffee"))
}

func (s *ServerURLSuite) TestServer_MakeAudioURL() {
	s.Require().Equal(
		"https://127.0.0.1:1337/messages/audio?messageId=0xde1e7ebee71e",
		s.server.MakeAudioURL("0xde1e7ebee71e"))
	s.Require().Equal(
		"https://127.0.0.1:0/messages/audio?messageId=0xde1e7ebee71e",
		s.serverNoPort.MakeAudioURL("0xde1e7ebee71e"))
}

func (s *ServerURLSuite) TestServer_MakeStickerURL() {
	s.Require().Equal(
		"https://127.0.0.1:1337/ipfs?hash=0xdeadbeef4ac0",
		s.server.MakeStickerURL("0xdeadbeef4ac0"))
	s.Require().Equal(
		"https://127.0.0.1:0/ipfs?hash=0xdeadbeef4ac0",
		s.serverNoPort.MakeStickerURL("0xdeadbeef4ac0"))
}
