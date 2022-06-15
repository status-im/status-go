package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"
	"testing"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stretchr/testify/suite"
)

func TestCerts(t *testing.T) {
	suite.Run(t, new(CertsSuite))
}

const (
	X    = "7744735542292224619198421067303535767629647588258222392379329927711683109548"
	Y    = "6855516769916529066379811647277920115118980625614889267697023742462401590771"
	D    = "38564357061962143106230288374146033267100509055924181407058066820384455255240"
	DB58 = "6jpbvo2ucrtrnpXXF4DQYuysh697isH9ppd2aT8uSRDh"
	SN   = "91849736469742262272885892667727604096707836853856473239722372976236128900962"
)

type CertsSuite struct {
	suite.Suite

	X      *big.Int
	Y      *big.Int
	D      *big.Int
	DBytes []byte
	SN     *big.Int
}

func (s *CertsSuite) SetupSuite() {
	var ok bool

	s.X, ok = new(big.Int).SetString(X, 10)
	s.Require().True(ok)

	s.Y, ok = new(big.Int).SetString(Y, 10)
	s.Require().True(ok)

	s.D, ok = new(big.Int).SetString(D, 10)
	s.Require().True(ok)

	s.DBytes = base58.Decode(DB58)
	s.Require().Exactly(s.D.Bytes(), s.DBytes)

	s.SN, ok = new(big.Int).SetString(SN, 10)
	s.Require().True(ok)
}

func (s *CertsSuite) Test_makeSerialNumberFromKey() {
	pk := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     s.X,
			Y:     s.Y,
		},
		D: s.D,
	}

	s.Require().Zero(makeSerialNumberFromKey(pk).Cmp(s.SN))
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

func (s *CertsSuite) TestGenerateX509Cert() {
	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour)

	c1 := GenerateX509Cert(s.SN, notBefore, notAfter, localhost)
	s.Require().Exactly([]string{localhost}, c1.DNSNames)
	s.Require().Nil(c1.IPAddresses)

	c2 := GenerateX509Cert(s.SN, notBefore, notAfter, defaultIP.String())
	s.Require().Len(c2.IPAddresses, 1)
	s.Require().Equal(defaultIP.String(), c2.IPAddresses[0].String())
	s.Require().Nil(c2.DNSNames)
}
