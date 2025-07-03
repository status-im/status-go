package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrBigIntSetFromString(t *testing.T) {
	tcs := []struct {
		Value    string
		Expected string
	}{
		{"hello", "failed to set big.Int balance from string 'hello'"},
		{"123456.44abc", "failed to set big.Int balance from string '123456.44abc'"},
		{"13e1234234", "failed to set big.Int balance from string '13e1234234'"},
	}

	for _, tc := range tcs {
		require.Equal(t, tc.Expected, ErrBigIntSetFromString(tc.Value).Error())
	}
}
