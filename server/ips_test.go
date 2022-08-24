package server

import (
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestGetOutboundIPSuite(t *testing.T) {
	suite.Run(t, new(GetOutboundIPSuite))
}

type GetOutboundIPSuite struct {
	suite.Suite
	TestPairingServerComponents
}

func (s *GetOutboundIPSuite) SetupSuite() {
	s.SetupPairingServerComponents(s.T())
}

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

func (s *GetOutboundIPSuite) TestGetOutboundIPWithFullServerE2e() {
	s.PS.mode = Sending
	s.PS.SetHandlers(HandlerPatternMap{"/hello": testHandler(s.T())})

	err := s.PS.Start()
	s.Require().NoError(err)

	// Give time for the sever to be ready, hacky I know, I'll iron this out
	time.Sleep(100 * time.Millisecond)

	// Server generates a QR code connection string
	cp, err := s.PS.MakeConnectionParams()
	s.Require().NoError(err)

	qr, err := cp.ToString()
	s.Require().NoError(err)

	// Client reads QR code and parses the connection string
	ccp := new(ConnectionParams)
	err = ccp.FromString(qr)
	s.Require().NoError(err)

	c, err := NewPairingClient(ccp, nil)
	s.Require().NoError(err)

	thing, err := makeThingToSay()
	s.Require().NoError(err)

	response, err := c.Get(c.baseAddress.String() + "/hello?say=" + thing)
	s.Require().NoError(err)

	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	s.Require().NoError(err)
	s.Require().Equal("Hello I like to be a tls server. You said: `"+thing+"`", string(content[:109]))
}
