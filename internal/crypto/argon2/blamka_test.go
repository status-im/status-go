package argon2

import (
	"crypto/rand"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func randomBlock() *block {
	var raw [8 * blockLength]byte
	_, _ = rand.Read(raw[:])
	var b block
	for i := range b {
		b[i] = binary.LittleEndian.Uint64(raw[i*8:])
	}
	return &b
}

func TestProcessBlockMatchesGeneric(t *testing.T) {
	for i := 0; i < 200; i++ {
		in1, in2, out := randomBlock(), randomBlock(), randomBlock()
		for _, xor := range []bool{false, true} {
			want, got := *out, *out
			processBlockGeneric(&want, in1, in2, xor)
			if xor {
				processBlockXOR(&got, in1, in2)
			} else {
				processBlock(&got, in1, in2)
			}
			require.Equal(t, want, got, "block %d xor %v", i, xor)
		}
	}
}

func BenchmarkProcessBlockXOR(b *testing.B) {
	in1, in2, out := randomBlock(), randomBlock(), randomBlock()
	b.SetBytes(int64(len(out) * 8))
	for b.Loop() {
		processBlockXOR(out, in1, in2)
	}
}

func BenchmarkProcessBlockXORGeneric(b *testing.B) {
	in1, in2, out := randomBlock(), randomBlock(), randomBlock()
	b.SetBytes(int64(len(out) * 8))
	for b.Loop() {
		processBlockGeneric(out, in1, in2, true)
	}
}
