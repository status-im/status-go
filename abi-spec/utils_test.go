package abispec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSha3(t *testing.T) {
	require.Equal(t, "48bed44d1bcd124a28c27f343a817e5f5243190d3c52bf347daf876de1dbbf77", Sha3("abcd"))
	require.Equal(t, "79bc958fa37445ba1b909c34a73cecfa4966a6e6783f90863dd6df5a80f96ad0", Sha3("12786e7b2111caae36dd91aca91ce627f26fa3c77018a98880ab50a82ac6b6aa835f6bbf96da54aa6b88ceffb8107ef40a4ef8a85bf4eb0e81a9464e0a27fcf3"))
	require.Equal(t, "04d874cdee68658bd64a07b6180dd36b735b60876ca79a29f88114bc78e6fc32", Sha3("0x12786e7b2111caae36dd91aca91ce627f26fa3c77018a98880ab50a82ac6b6aa835f6bbf96da54aa6b88ceffb8107ef40a4ef8a85bf4eb0e81a9464e0a27fcf3"))
	require.Equal(t, "04d874cdee68658bd64a07b6180dd36b735b60876ca79a29f88114bc78e6fc32", Sha3("0x12786E7b2111caae36dd91aca91ce627f26fa3c77018a98880ab50a82ac6b6aa835f6bbf96da54aa6b88ceffb8107ef40a4ef8a85bf4eb0e81a9464e0a27fcf3"))
}

const (
	address1 = "0x0eD343df16A5327aC13B689072804A2705a26F47"
	address2 = "0x0eD343df16A5327aC13B689072804A2705a26F47a"
	address3 = "0x0eD343df16A5327aC13B689072804A2705a26f47"
)

func TestIsAddress(t *testing.T) {
	valid, err := IsAddress(address1)
	require.NoError(t, err)
	require.True(t, valid)

	valid, err = IsAddress(address2)
	require.NoError(t, err)
	require.False(t, valid)

	valid, err = IsAddress(address3)
	require.NoError(t, err)
	require.False(t, valid)
}

func TestCheckAddressChecksum(t *testing.T) {
	valid, err := CheckAddressChecksum(address1)
	require.NoError(t, err)
	require.True(t, valid)

	valid, err = CheckAddressChecksum(address2)
	require.NoError(t, err)
	require.False(t, valid)

	valid, err = CheckAddressChecksum(address3)
	require.NoError(t, err)
	require.False(t, valid)
}
