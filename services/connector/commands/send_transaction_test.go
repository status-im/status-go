package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionCommandExecution(t *testing.T) {
	cmd := &SendTransactionCommand{}

	request := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_sendTransaction",
		Params:  []interface{}{map[string]string{"from": "0x1234", "to": "0x5678", "value": "0x1"}},
	}
	expectedOutput := "transaction sent"

	result, err := cmd.Execute(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, result)
}
