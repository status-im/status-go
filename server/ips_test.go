package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	t.Skip("Don't run in CI ... yet")

	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	ip, _ := GetOutboundIP()
	spew.Dump(ip.String())

	cert, err := GenerateCertFromKey(pk, time.Hour, ip)
	require.NoError(t, err)

	s, err := NewServer(nil, nil, SetCert(&cert), SetNetIP(ip), SetPort(8088))
	require.NoError(t, err)

	s.WithHandlers(HandlerPatternMap{"/hello": testHandler})

	s.listenAndServe()
}
