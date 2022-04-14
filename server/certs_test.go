package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_makeSerialNumberFromKey(t *testing.T) {
	x, ok := new(big.Int).SetString("7744735542292224619198421067303535767629647588258222392379329927711683109548", 10)
	require.True(t, ok)

	y, ok := new(big.Int).SetString("6855516769916529066379811647277920115118980625614889267697023742462401590771", 10)
	require.True(t, ok)

	d, ok := new(big.Int).SetString("38564357061962143106230288374146033267100509055924181407058066820384455255240", 10)
	require.True(t, ok)

	testPk := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X: x,
			Y: y,
		},
		D: d,
	}

	sn := makeSerialNumberFromKey(testPk)
	tsn, ok := new(big.Int).SetString("91849736469742262272885892667727604096707836853856473239722372976236128900962", 10)
	require.True(t, ok)

	require.Equal(t, 0, sn.Cmp(tsn))
}
