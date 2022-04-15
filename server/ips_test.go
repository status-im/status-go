package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	cert, err := GenerateCertFromKey(pk, time.Hour)
	require.NoError(t, err)

	s, err := NewServer(nil, nil, SetCert(&cert), SetPort(8088))
	require.NoError(t, err)

	s.WithHandlers(HandlerPatternMap{"/hello": testHandler})

	spew.Dump(GetOutboundIP())

	s.listenAndServe()

	// TODO fix the 400 error ...
}

func TestSimple(t *testing.T) {
	http.HandleFunc("/hello", testHandler)

	http.ListenAndServe(":8090", nil)
}

func TestOld(t *testing.T) {
	s, err := NewServer(nil, nil)
	require.NoError(t, err)

	s.WithMediaHandlers()
	s.WithHandlers(HandlerPatternMap{"/hello": testHandler})
	s.listenAndServe()
}