package argon2

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	xargon2 "golang.org/x/crypto/argon2"
)

type params struct {
	time, memory uint32
	threads      uint8
	keyLen       uint32
}

var paramSets = []params{
	{time: 1, memory: 64, threads: 1, keyLen: 32},
	{time: 1, memory: 1024, threads: 4, keyLen: 32},
	{time: 3, memory: 1024, threads: 4, keyLen: 32},
	{time: 2, memory: 333, threads: 3, keyLen: 100},
	{time: 1, memory: 8, threads: 1, keyLen: 16},
	{time: 1, memory: 4096, threads: 2, keyLen: 64},
}

func TestHasherMatchesXCrypto(t *testing.T) {
	var h Hasher
	salt := []byte("somesalt12345678")
	// One hasher across every params set and nonce: stale blocks from earlier derivations must not leak.
	for round := 0; round < 2; round++ {
		for _, p := range paramSets {
			for nonce := 0; nonce < 4; nonce++ {
				password := []byte(fmt.Sprintf("challenge%d", nonce))
				want := xargon2.IDKey(password, salt, p.time, p.memory, p.threads, p.keyLen)
				got := h.IDKey(password, salt, p.time, p.memory, p.threads, p.keyLen)
				require.Equal(t, want, got, "params %+v nonce %d", p, nonce)
			}
		}
	}
}

func TestHasherReusesBlocks(t *testing.T) {
	const memory = 4096
	var h Hasher
	h.IDKey([]byte("p"), []byte("saltsalt"), 1, memory, 1, 32)
	var fresh Hasher
	freshBytes := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			fresh.Release()
			fresh.IDKey([]byte("p"), []byte("saltsalt"), 1, memory, 1, 32)
		}
	}).AllocedBytesPerOp()
	reusedBytes := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			h.IDKey([]byte("p"), []byte("saltsalt"), 1, memory, 1, 32)
		}
	}).AllocedBytesPerOp()
	require.GreaterOrEqual(t, freshBytes, int64(memory*1024))
	require.Less(t, reusedBytes, int64(64*1024), "reused derivation allocated %d bytes", reusedBytes)
}

func BenchmarkXCryptoIDKey(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		xargon2.IDKey([]byte("p"), []byte("saltsalt"), 1, 8*1024, 1, 32)
	}
}

func BenchmarkHasherIDKey(b *testing.B) {
	var h Hasher
	b.ReportAllocs()
	for b.Loop() {
		h.IDKey([]byte("p"), []byte("saltsalt"), 1, 8*1024, 1, 32)
	}
}
