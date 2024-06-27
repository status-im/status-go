package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignCommandExecution(t *testing.T) {
	cmd := &SignCommand{}

	request := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_sign",
		Params:  []interface{}{"0x1234", "0x5678"},
	}
	expectedOutput := "signed"

	result, err := cmd.Execute(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, result)
}
