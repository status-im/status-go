package pairing

import (
	"testing"

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

func (s *CertsSuite) Test_makeSerialNumberFromKey() {
	s.Require().Zero(makeSerialNumberFromKey(s.PK).Cmp(s.SN))
}
