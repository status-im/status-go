package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
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

func makeThingToSay() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
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

	s := NewPairingServer(&Config{pk, &cert, ip.String(), Sending})

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

	c, err := NewClient(ccp)
	require.NoError(t, err)

	thing, err := makeThingToSay()
	require.NoError(t, err)

	response, err := c.Get(c.baseAddress.String() + "/hello?say=" + thing)
	require.NoError(t, err)

	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, "Hello I like to be a tls server. You said: `"+thing+"`", string(content[:109]))
}
