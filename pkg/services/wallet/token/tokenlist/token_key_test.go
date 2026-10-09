package tokenlist

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func referenceTokenKey(chainID uint64, address common.Address) string {
	return fmt.Sprintf("%d-%s", chainID, strings.ToLower(address.Hex()))
}

func TestTokenKeyFormat(t *testing.T) {
	for _, chainID := range []uint64{0, 1, 56, 8453, 59144, ^uint64(0)} {
		for _, address := range []common.Address{
			{},
			common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
			common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff"),
		} {
			require.Equal(t, referenceTokenKey(chainID, address), TokenKey(chainID, address))
		}
	}
}

func BenchmarkTokenKey(b *testing.B) {
	address := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	b.ReportAllocs()
	for b.Loop() {
		_ = TokenKey(8453, address)
	}
}

func FuzzTokenKeyFormat(f *testing.F) {
	f.Add(uint64(1), []byte{0xa0, 0xb8, 0x69, 0x91})
	f.Fuzz(func(t *testing.T, chainID uint64, b []byte) {
		address := common.BytesToAddress(b)
		require.Equal(t, referenceTokenKey(chainID, address), TokenKey(chainID, address))
	})
}
