package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type nonJSON struct{}

func (*nonJSON) MarshalJSON() ([]byte, error) {
	return nil, errors.New("invalid JSON")
}

func TestPrepareJSONResponseErrorWithResult(t *testing.T) {
	var data []byte

	data = prepareJSONResponse("0x123", nil)
	require.Equal(t, `{"result":"0x123"}`, string(data))

	data = prepareJSONResponse(&nonJSON{}, nil)
	require.Contains(t, string(data), `{"error":{"message":`)
}

func TestPrepareJSONResponseErrorWithError(t *testing.T) {
	data := prepareJSONResponse("0x123", errors.New("some error"))
	require.Contains(t, string(data), `{"error":{"message":"some error"}}`)
}
