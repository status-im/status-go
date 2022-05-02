package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testHandler(t *testing.T) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		say, ok := r.URL.Query()["say"]
		if !ok || len(say) == 0 {
			say = append(say, "nothing")
		}

		_, err := w.Write([]byte("Hello I like to be a tls server. You said: `" + say[0] + "` " + time.Now().String()))
		if err != nil {
			require.NoError(t, err)
		}
	}
}

func TestGetOutboundIPWithFullServerE2e(t *testing.T) {
	// Get 3 key components for tls.cert generation
	// 1) Ephemeral private key
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// 2) Device outbound IP address
	ip, err := GetOutboundIP()
	require.NoError(t, err)

	// 3) NotBefore time
	certTime := time.Now()

	// Generate tls.Certificate and Server
	cert, _, err := GenerateCertFromKey(pk, certTime, ip.String())
	require.NoError(t, err)

	s := NewPairingServer(&Config{pk, &cert, ip.String()})

	s.SetHandlers(HandlerPatternMap{"/hello": testHandler(t)})

	err = s.Start()
	require.NoError(t, err)

	// Give time for the sever to be ready, hacky I know, I'll iron this out
	time.Sleep(100 * time.Millisecond)

	// Server generates a QR code connection string
	cp, err := s.MakeConnectionParams()
	require.NoError(t, err)

	qr, err := cp.ToString()
	require.NoError(t, err)

	// Client reads QR code and parses the connection string
	ccp := new(ConnectionParams)
	err = ccp.FromString(qr)
	require.NoError(t, err)

	u, certPem, err := ccp.Generate()
	require.NoError(t, err)

	rootCAs, err := x509.SystemCertPool()
	require.NoError(t, err)

	ok := rootCAs.AppendCertsFromPEM(certPem)
	require.True(t, ok)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: false, // MUST BE FALSE, or the test is meaningless
			RootCAs:            rootCAs,
		},
	}
	client := &http.Client{Transport: tr}

	b := make([]byte, 32)
	_, err = rand.Read(b)
	require.NoError(t, err)
	thing := hex.EncodeToString(b)

	response, err := client.Get(u.String() + "/hello?say=" + thing)
	require.NoError(t, err)

	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, "Hello I like to be a tls server. You said: `"+thing+"`", string(content[:109]))
}
