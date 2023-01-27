package statusgo

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
	data := prepareJSONResponse("0x123", nil)
	require.Equal(t, `{"result":"0x123"}`, data)

	data = prepareJSONResponse(&nonJSON{}, nil)
	require.Contains(t, data, `{"error":{"code":1,"message":`)
}

func TestPrepareJSONResponseErrorWithError(t *testing.T) {
	data := prepareJSONResponse("0x123", errors.New("some error"))
	require.Contains(t, data, `{"error":{"message":"some error"}}`)
}

func TestDeserializeAndCompressKeyApi(t *testing.T) {
	desktopKey := "zQ3shTAten2v9CwyQD1Kc7VXAqNPDcHZAMsfbLHCZEx6nFqk9"
	mobileKeyExpected := "0x025596a7ff87da36860a84b0908191ce60a504afc94aac93c1abd774f182967ce6"
	mobileKeyConverted := DeserializeAndCompressKey(desktopKey)
	require.Equal(t, mobileKeyConverted, mobileKeyExpected)
}
