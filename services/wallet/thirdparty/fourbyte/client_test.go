package fourbyte

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	client := NewClient()
	res, err := client.Run("0x40e8d703000000000000000000000000670dca62b3418bddd08cbc69cb4490a5a3382a9f0000000000000000000000000000000000000000000000000000000000000064")
	require.Nil(t, err)
	require.Equal(t, res.Signature, "processDepositQueue(address,uint256)")
	require.Equal(t, res.Name, "processDepositQueue")
	require.Equal(t, res.ID, "0xf94d2")
	require.Equal(t, res.Inputs, map[string]string{
		"0": "0x3030303030303030303030303637306463613632",
		"1": "44417128579249187980157595307322491418158007948522794164811090501355597543782",
	})

	_, err = client.Run("0x70a08231000")
	require.NotNil(t, err)

	_, err = client.Run("0x70a082310")
	require.NotNil(t, err)
}
