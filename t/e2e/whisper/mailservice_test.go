package whisper

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/t/e2e"
	"github.com/stretchr/testify/suite"
)

func TestMailServiceSuite(t *testing.T) {
	suite.Run(t, new(MailServiceSuite))
}

type MailServiceSuite struct {
	e2e.StatusNodeTestSuite
}

func (s *MailServiceSuite) SetupTest() {
	s.StatusNode = node.New()
}

// TestShhextRequestMessagesRPCMethodAvailability tests if `shhext_requestMessages` is available
// through inproc and HTTP interfaces.
func (s *MailServiceSuite) TestShhextRequestMessagesRPCMethodAvailability() {
	r := s.Require()

	s.StartTestNode(func(config *params.NodeConfig) {
		config.RPCEnabled = true
	})
	defer s.StopTestNode()

	client := s.StatusNode.RPCClient()
	r.NotNil(client)

	// This error means that the method is available through inproc communication
	// as the validation of params occurred.
	err := client.Call(nil, "shhext_requestMessages", map[string]interface{}{})
	r.EqualError(err, `invalid mailServerPeer value: invalid URL scheme, want "enode"`)

	// Do the same but using HTTP interface.
	req, err := http.NewRequest("POST", "http://localhost:8645", bytes.NewBuffer([]byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "shhext_requestMessages",
		"params": [{}]
	}`)))
	req.Header.Set("Content-Type", "application/json")
	r.NoError(err)
	resp, err := http.DefaultClient.Do(req)
	r.NoError(err)
	defer func() {
		err := resp.Body.Close()
		r.NoError(err)
	}()
	r.Equal(200, resp.StatusCode)
	data, err := ioutil.ReadAll(resp.Body)
	r.NoError(err)
	r.Contains(string(data), `invalid mailServerPeer value`)
}
