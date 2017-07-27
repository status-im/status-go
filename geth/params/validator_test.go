package params

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNetworkValidator(t *testing.T) {
	type TestStruct struct {
		TestField uint64 `validate:"network"`
	}

	testStruct := TestStruct{TestField: 0}

	validate := NewValidator()

	err := validate.Struct(&testStruct)
	// err is "Key: 'TestStruct.TestField' Error:Field validation for 'TestField' failed on the 'network' tag"
	require.Error(t, err)
}
