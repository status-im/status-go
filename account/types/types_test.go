package types

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/crypto"
	ethtypes "github.com/status-im/status-go/eth-node/types"
)

func TestHex(t *testing.T) {
	var addr *SelectedExtKey
	cr, _ := crypto.GenerateKey()
	var flagtests = []struct {
		in  *SelectedExtKey
		out string
	}{
		{&SelectedExtKey{
			Address:    ethtypes.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e"),
			AccountKey: &ethtypes.Key{PrivateKey: cr},
		}, "0x742d35Cc6634C0532925a3b844Bc454e4438f44e"},
		{addr, "0x0"},
	}

	for _, tt := range flagtests {
		assert.Equal(t, tt.in.Hex(), tt.out)
	}
}
