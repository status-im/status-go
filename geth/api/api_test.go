package api

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/status-im/status-go/geth/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNodeConfig(t *testing.T) {
	noErrorsCallback := func(resp common.APIDetailedResponse) {
		require.True(t, resp.Status, "expected status equal true")
		require.Empty(t, resp.FieldErrors)
		require.Empty(t, resp.Message)
	}

	testCases := []struct {
		Name     string
		Config   string
		Callback func(common.APIDetailedResponse)
	}{
		{
			Name: "response for valid config",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/tmp"
			}`,
			Callback: noErrorsCallback,
		},
		{
			Name:   "response for invalid JSON string",
			Config: `{"Network": }`,
			Callback: func(resp common.APIDetailedResponse) {
				require.False(t, resp.Status)
				require.Contains(t, resp.Message, "validation: invalid character '}'")
			},
		},
		{
			Name:   "response for config with multiple errors",
			Config: `{}`,
			Callback: func(resp common.APIDetailedResponse) {
				required := map[string]string{
					"NodeConfig.NetworkID": "required",
					"NodeConfig.DataDir":   "required",
				}

				require.False(t, resp.Status)
				require.Contains(t, resp.Message, "validation: validation failed")
				require.Equal(t, 2, len(resp.FieldErrors))

				for _, err := range resp.FieldErrors {
					require.Contains(t, required, err.Parameter)
					require.Contains(t, err.Error(), required[err.Parameter])
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Logf("TestValidateNodeConfig: %s", tc.Name)
		testValidateNodeConfig(tc.Config, tc.Callback)
	}
}

func testValidateNodeConfig(config string, fn func(common.APIDetailedResponse)) {
	statusAPI := StatusAPI{}
	resp := statusAPI.ValidateJSONConfig(config)
	fn(resp)
}

func Test_processError(t *testing.T) {
	capture := StderrCapture{}
	tests := []struct {
		name       string
		err        error
		want       string
		wantStdErr string
	}{
		{"nilError", nil, "", ""},
		{"notNilError", fmt.Errorf("test error"), "test error", "test error\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture.StartCapture()
			if got := processError(tt.err); got != tt.want {
				assert.Equal(t, tt.want, got, "Error message did not match")
			}
			stdErr, err := capture.StopCapture()
			assert.NoError(t, err, "Got an error trying to capture stdout and stderr!")
			assert.Equal(t, tt.wantStdErr, stdErr, "StdErr did not match expected output")

		})
	}
}

type StderrCapture struct {
	oldStderr *os.File
	readPipe  *os.File
}

func (sc *StderrCapture) StartCapture() {
	sc.oldStderr = os.Stderr
	sc.readPipe, os.Stderr, _ = os.Pipe()
}

func (sc *StderrCapture) StopCapture() (string, error) {
	if sc.oldStderr == nil || sc.readPipe == nil {
		return "", errors.New("StartCapture not called before StopCapture")
	}
	err := os.Stderr.Close()
	if err != nil {
		return "", err
	}
	os.Stderr = sc.oldStderr
	bytes, err := ioutil.ReadAll(sc.readPipe)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
