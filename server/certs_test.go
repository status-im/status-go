package server

import (
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stretchr/testify/suite"
)

func TestCerts(t *testing.T) {
	suite.Run(t, new(CertsSuite))
}

type CertsSuite struct {
	suite.Suite
	TestKeyComponents
	TestCertComponents
}

func (s *CertsSuite) SetupSuite() {
	s.SetupKeyComponents(s.T())
	s.SetupCertComponents(s.T())
}

func (s *CertsSuite) Test_makeSerialNumberFromKey() {
	s.Require().Zero(makeSerialNumberFromKey(s.PK).Cmp(s.SN))
}

func (s *CertsSuite) TestToECDSA() {
	k := ToECDSA(base58.Decode(DB58))
	s.Require().NotNil(k.PublicKey.X)
	s.Require().NotNil(k.PublicKey.Y)

	s.Require().Zero(k.PublicKey.X.Cmp(s.X))
	s.Require().Zero(k.PublicKey.Y.Cmp(s.Y))
	s.Require().Zero(k.D.Cmp(s.D))

	b58 := base58.Encode(s.D.Bytes())
	s.Require().Equal(DB58, b58)
}
