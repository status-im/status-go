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

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func testHandler(w http.ResponseWriter, r *http.Request) {
	say, ok := r.URL.Query()["say"]
	if !ok || len(say) == 0 {
		say = append(say, "nothing")
	}

	_, err := w.Write([]byte("Hello I like to be a tls server. You said: `" + say[0] + "` " + time.Now().String()))
	if err != nil {
		// Dump err this is only a testHandler
		spew.Dump(err)
	}
}

func TestGetOutboundIPWithFullServerE2e(t *testing.T) {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	ip, _ := GetOutboundIP()
	spew.Dump(ip.String())

	cert, certPem, err := GenerateCertFromKey(pk, time.Hour, ip)
	require.NoError(t, err)

	s, err := NewServer(nil, nil, &Config{&cert, ip, 8088})
	require.NoError(t, err)

	s.WithHandlers(HandlerPatternMap{"/hello": testHandler})

	go s.listenAndServe()

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

	response, err := client.Get(s.MakeBaseURL().String() + "/hello?say=" + thing)
	require.NoError(t, err)

	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, "Hello I like to be a tls server. You said: `"+thing+"`", string(content[:109]))
}
