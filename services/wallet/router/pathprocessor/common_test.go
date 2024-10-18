package pathprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGettingNameFromEnsUsername(t *testing.T) {
	ensName := "test"
	name := getNameFromEnsUsername(ensName)
	require.Equal(t, ensName, name)

	ensStatusName := "test.stateofus.eth"
	name = getNameFromEnsUsername(ensStatusName)
	require.Equal(t, ensName, name)

	ensNotStatusName := "test.eth"
	name = getNameFromEnsUsername(ensNotStatusName)
	require.Equal(t, ensNotStatusName, name)
}
