package lightwallet

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/text/unicode/norm"
)

const (
	pairingTokenSalt = "Status Hardware Wallet Lite"
	maxPukNumber     = int64(999999999999)
)

type Secrets struct {
	puk          string
	pairingPass  string
	pairingToken []byte
}

func NewSecrets() (*Secrets, error) {
	pairingPass, err := generatePairingPass()
	if err != nil {
		return nil, err
	}

	puk, err := rand.Int(rand.Reader, big.NewInt(maxPukNumber))
	if err != nil {
		return nil, err
	}

	return &Secrets{
		puk:          fmt.Sprintf("%012d", puk.Int64()),
		pairingPass:  pairingPass,
		pairingToken: generatePairingToken(pairingPass),
	}, nil
}

func (s *Secrets) Puk() string {
	return s.puk
}

func (s *Secrets) PairingPass() string {
	return s.pairingPass
}

func (s *Secrets) PairingToken() []byte {
	return s.pairingToken
}

func generatePairingPass() (string, error) {
	r := make([]byte, 12)
	_, err := rand.Read(r)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(r), nil
}

func generatePairingToken(pass string) []byte {
	return pbkdf2.Key(norm.NFKD.Bytes([]byte(pass)), norm.NFKD.Bytes([]byte(pairingTokenSalt)), 50000, 32, sha256.New)
}
