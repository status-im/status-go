package server

import (
	"testing"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/server/servertest"
)

func TestCerts(t *testing.T) {
	suite.Run(t, new(CertsSuite))
}

type CertsSuite struct {
	suite.Suite
	servertest.TestKeyComponents
	servertest.TestCertComponents
}

func (s *CertsSuite) SetupSuite() {
	s.SetupKeyComponents(s.T())
	s.SetupCertComponents(s.T())
}

func (s *CertsSuite) TestToECDSA() {
	k := ToECDSA(base58.Decode(servertest.DB58))
	s.Require().NotNil(k.PublicKey.X)
	s.Require().NotNil(k.PublicKey.Y)

	s.Require().Zero(k.PublicKey.X.Cmp(s.X))
	s.Require().Zero(k.PublicKey.Y.Cmp(s.Y))
	s.Require().Zero(k.D.Cmp(s.D))

	b58 := base58.Encode(s.D.Bytes())
	s.Require().Equal(servertest.DB58, b58)
}

func (s *CertsSuite) TestGenerateX509Cert() {
	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour)

	c1 := GenerateX509Cert(s.SN, notBefore, notAfter, Localhost)
	s.Require().Exactly([]string{Localhost}, c1.DNSNames)
	s.Require().Nil(c1.IPAddresses)

	c2 := GenerateX509Cert(s.SN, notBefore, notAfter, DefaultIP.String())
	s.Require().Len(c2.IPAddresses, 1)
	s.Require().Equal(DefaultIP.String(), c2.IPAddresses[0].String())
	s.Require().Nil(c2.DNSNames)
}
