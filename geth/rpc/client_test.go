package rpc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetResultFromRPCResponse(t *testing.T) {
	var err error

	var resultRawMessage json.RawMessage
	err = setResultFromRPCResponse(&resultRawMessage, []string{"one", "two", "three"})
	require.NoError(t, err)
	require.Equal(t, json.RawMessage(`["one","two","three"]`), resultRawMessage)

	var resultSlice []int
	err = setResultFromRPCResponse(&resultSlice, []int{1, 2, 3})
	require.NoError(t, err)
	require.Equal(t, []int{1, 2, 3}, resultSlice)

	var resultMap map[string]interface{}
	err = setResultFromRPCResponse(&resultMap, map[string]interface{}{"test": true})
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"test": true}, resultMap)

	var resultStruct struct {
		A int
		B string
	}
	err = setResultFromRPCResponse(&resultStruct, struct {
		A int
		B string
	}{5, "test"})
	require.NoError(t, err)
	require.Equal(t, struct {
		A int
		B string
	}{5, "test"}, resultStruct)

	var resultIncorrectType []int
	err = setResultFromRPCResponse(&resultIncorrectType, []string{"a", "b"})
	require.Error(t, err)
}
