package api

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
)

func TestCreateAccountAndLogin(t *testing.T) {
	rootDir := t.TempDir()
	t.Logf("TestCreateAccountAndLogin: rootDir: %s", rootDir)

	requestJSONTemplateString := `
{
    "upstreamConfig":"",
    "openseaAPIKey":"",
    "wakuV2Nameserver":"1.1.1.1",
    "mnemonic":null,
    "verifyENSContractAddress":"0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e",
    "rootDataDir":"%s",
    "password":"0x20756cad9b728c8225fd8cedb6badaf8731e174506950219ea657cd54f35f46c",
    "displayName":"%s",
    "logEnabled":true,
    "verifyTransactionChainID":1,
    "currentNetwork":"mainnet_rpc",
    "customizationColor":"blue",
	"emoji": "some",
    "previewPrivacy":true,
    "verifyTransactionURL":"",
    "imagePath":null,
    "verifyENSURL":"",
    "logLevel":"INFO",
    "logFilePath":"%s",
    "networkId":1
}
`
	requestJSON := fmt.Sprintf(requestJSONTemplateString, rootDir, "user1", rootDir)
	var request requests.CreateAccount
	err := json.Unmarshal([]byte(requestJSON), &request)
	require.NoError(t, err)
	statusBackend := NewGethStatusBackend(tt.MustCreateTestLogger())
	_, err = statusBackend.CreateAccountAndLogin(&request)
	require.NoError(t, err)
	t.Logf("TestCreateAccountAndLogin: create account user1 and login successfully")
	// wait waku node start working
	time.Sleep(2 * time.Second)

	t.Logf("TestCreateAccountAndLogin: logouting")
	err = statusBackend.Logout()
	require.NoError(t, err)
	t.Logf("TestCreateAccountAndLogin: logout done")

	requestJSON = fmt.Sprintf(requestJSONTemplateString, rootDir, "user2", rootDir)
	err = json.Unmarshal([]byte(requestJSON), &request)
	require.NoError(t, err)
	_, err = statusBackend.CreateAccountAndLogin(&request)
	require.NoError(t, err)
}
