// Package servertest provides utilities for server testing.
package servertest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/asn1"
	"github.com/status-im/status-go/logutils"
	"math/big"
	"testing"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	X        = "7744735542292224619198421067303535767629647588258222392379329927711683109548"
	Y        = "6855516769916529066379811647277920115118980625614889267697023742462401590771"
	D        = "38564357061962143106230288374146033267100509055924181407058066820384455255240"
	AES      = "BbnZ7Gc66t54a9kEFCf7FW8SGQuYypwHVeNkRYeNoqV6"
	DB58     = "6jpbvo2ucrtrnpXXF4DQYuysh697isH9ppd2aT8uSRDh"
	SN       = "91849736469742262272885892667727604096707836853856473239722372976236128900962"
	CertTime = "eQUriVtGtkWhPJFeLZjF"
)

type TestKeyComponents struct {
	X      *big.Int
	Y      *big.Int
	D      *big.Int
	AES    []byte
	DBytes []byte
	PK     *ecdsa.PrivateKey
}

func (tk *TestKeyComponents) SetupKeyComponents(t *testing.T) {
	var ok bool

	tk.X, ok = new(big.Int).SetString(X, 10)
	require.True(t, ok)

	tk.Y, ok = new(big.Int).SetString(Y, 10)
	require.True(t, ok)

	tk.D, ok = new(big.Int).SetString(D, 10)
	require.True(t, ok)

	tk.AES = base58.Decode(AES)
	require.Len(t, tk.AES, 32)

	tk.DBytes = base58.Decode(DB58)
	require.Exactly(t, tk.D.Bytes(), tk.DBytes)

	tk.PK = &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     tk.X,
			Y:     tk.Y,
		},
		D: tk.D,
	}
}

type TestCertComponents struct {
	NotBefore, NotAfter time.Time
	SN                  *big.Int
}

func (tcc *TestCertComponents) SetupCertComponents(t *testing.T) {
	var ok bool

	tcc.SN, ok = new(big.Int).SetString(SN, 10)
	require.True(t, ok)

	_, err := asn1.Unmarshal(base58.Decode(CertTime), &tcc.NotBefore)
	require.NoError(t, err)

	tcc.NotAfter = tcc.NotBefore.Add(time.Hour)
}

type TestLoggerComponents struct {
	Logger *zap.Logger
}

func (tlc *TestLoggerComponents) SetupLoggerComponents() {
	tlc.Logger = logutils.ZapLogger()
}
