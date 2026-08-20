package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGettingNameFromEnsUsername(t *testing.T) {
	ensName := "test"
	name := GetNameFromEnsUsername(ensName)
	require.Equal(t, ensName, name)

	ensStatusName := "test.stateofus.eth"
	name = GetNameFromEnsUsername(ensStatusName)
	require.Equal(t, ensName, name)

	ensNotStatusName := "test.eth"
	name = GetNameFromEnsUsername(ensNotStatusName)
	require.Equal(t, ensNotStatusName, name)
}
