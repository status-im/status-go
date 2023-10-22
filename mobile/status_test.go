package statusgo

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateAccountAndLogin(t *testing.T) {
	requestObj := map[string]interface{}{
		"upstreamConfig":              "https://eth-archival.gateway.pokt.network/v1/lb/3ef2018191814b7e1009b8d9",
		"openseaAPIKey":               "",
		"wakuV2Nameserver":            "8.8.8.8",
		"poktToken":                   "3ef2018191814b7e1009b8d9",
		"verifyENSContractAddress":    "0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e",
		"deviceName":                  "sdk_gphone64_arm64 goldfish_arm64",
		"backupDisabledDataDir":       "no_backup",
		"password":                    "0x0d45e19766c0cadfe3af48b801102a9de4337ee41088e3561d9f1e9897aeeeae",
		"displayName":                 "dfefwefwef",
		"logEnabled":                  true,
		"alchemyArbitrumMainnetToken": "",
		"verifyTransactionChainID":    1,
		"currentNetwork":              "mainnet_rpc",
		"customizationColor":          "sky",
		"wakuV2LightClient":           true,
		"previewPrivacy":              true,
		"verifyTransactionURL":        "https://eth-archival.gateway.pokt.network/v1/lb/3ef2018191814b7e1009b8d9",
		"imagePath":                   nil,
		"alchemyOptimismMainnetToken": "",
		"alchemyArbitrumGoerliToken":  "",
		"verifyENSURL":                "https://eth-archival.gateway.pokt.network/v1/lb/3ef2018191814b7e1009b8d9",
		"rootKeystoreDir":             "/no_backup/keystore",
		"logLevel":                    "INFO",
		"infuraToken":                 "",
		"logFilePath":                 "/storage/emulated/0/Android/data/im.status.ethereum.debug/files/Download",
		"alchemyOptimismGoerliToken":  "",
		"networkId":                   1,
	}

	requestJSONBytes, err := json.Marshal(requestObj)
	assert.Nil(t, err, "Expected valid request JSON")
	requestJSON := string(requestJSONBytes)

	response := CreateAccountAndLogin(requestJSON)

	var apiResponse APIResponse
	err = json.Unmarshal([]byte(response), &apiResponse)
	assert.Nil(t, err, "Expected valid JSON response")

	// Check if the error in the response is either empty (success case) or a valid string (error case)
	assert.True(t, apiResponse.Error == "" || len(apiResponse.Error) > 0, "Expected either empty error or valid error string")
}
