package server

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"net"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/suite"
)

var (
	baseRegex = "https:\\/\\/127\\.0\\.0\\.1:[0-9]{2,5}"
)

func TestServerURLSuite(t *testing.T) {
	suite.Run(t, new(ServerURLSuite))
}

type ServerURLSuite struct {
	suite.Suite
	TestKeyComponents
	TestCertComponents

	server           *Server
	serverNoListener *Server
}

func (s *ServerURLSuite) SetupSuite() {
	s.SetupKeyComponents(s.T())
	s.SetupCertComponents(s.T())

	l, err := net.Listen("tcp", defaultIP.String()+":0")
	s.Require().NoError(err)

	cert, _, err := GenerateCertFromKey(s.PK, s.NotBefore, defaultIP)
	s.Require().NoError(err)

	s.server = &Server{
		pk:       s.PK,
		cert:     &cert,
		netIP:    defaultIP,
		listener: l,
	}
	s.serverNoListener = &Server{
		netIP: defaultIP,
	}
}

func (s *ServerURLSuite) TestServer_MakeQRData() {
	qr, err := s.server.MakeQRData()
	s.Require().NoError(err)

	s.Require().Regexp(
		"4FHRnp:[1-9|A-Z|a-z]{1,4}:6jpbvo2ucrtrnpXXF4DQYuysh697isH9ppd2aT8uSRDh:eQUriVtGtkWhPJFeLZjF",
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
	s.Require().Regexp(baseRegex, s.server.MakeBaseURL().String())
	s.Require().Equal("https://127.0.0.1:0", s.serverNoListener.MakeBaseURL().String())
}

func (s *ServerURLSuite) TestServer_MakeImageServerURL() {
	s.Require().Regexp(baseRegex+"\\/messages\\/", s.server.MakeImageServerURL())
	s.Require().Equal("https://127.0.0.1:0/messages/", s.serverNoListener.MakeImageServerURL())
}

func (s *ServerURLSuite) TestServer_MakeIdenticonURL() {
	s.Require().Regexp(
		baseRegex+"\\/messages\\/identicons\\?publicKey=0xdaff0d11decade",
		s.server.MakeIdenticonURL("0xdaff0d11decade"))
	s.Require().Equal(
		"https://127.0.0.1:0/messages/identicons?publicKey=0xdaff0d11decade",
		s.serverNoListener.MakeIdenticonURL("0xdaff0d11decade"))
}

func (s *ServerURLSuite) TestServer_MakeImageURL() {
	s.Require().Regexp(
		baseRegex+"\\/messages\\/images\\?messageId=0x10aded70ffee",
		s.server.MakeImageURL("0x10aded70ffee"))
	s.Require().Equal(
		"https://127.0.0.1:0/messages/images?messageId=0x10aded70ffee",
		s.serverNoListener.MakeImageURL("0x10aded70ffee"))
}

func (s *ServerURLSuite) TestServer_MakeAudioURL() {
	s.Require().Regexp(
		baseRegex+"\\/messages\\/audio\\?messageId=0xde1e7ebee71e",
		s.server.MakeAudioURL("0xde1e7ebee71e"))
	s.Require().Equal(
		"https://127.0.0.1:0/messages/audio?messageId=0xde1e7ebee71e",
		s.serverNoListener.MakeAudioURL("0xde1e7ebee71e"))
}
