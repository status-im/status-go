package puzzleauth

import (
	"context"
	"encoding/hex"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// Unsolvable puzzle: every attempt runs, so allocations scale with the attempt count unless
// the argon2 block memory is reused across nonces.
func unsolvablePuzzle(memoryKB int) *Puzzle {
	return &Puzzle{
		Challenge:    "challenge",
		Salt:         hex.EncodeToString([]byte("salt12345678")),
		Difficulty:   64,
		Argon2Params: Argon2Params{MemoryKB: memoryKB, Time: 1, Threads: 1, KeyLen: 32},
	}
}

func TestSolve_ReusesBlockMemoryAcrossNonces(t *testing.T) {
	const memoryKB = 8 * 1024
	const attempts = 20

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := solve(context.Background(), unsolvablePuzzle(memoryKB), attempts)
	runtime.ReadMemStats(&after)
	require.Error(t, err)

	allocated := after.TotalAlloc - before.TotalAlloc
	require.Less(t, allocated, uint64(2*memoryKB*1024),
		"%d attempts allocated %d MiB; block memory must be allocated once", attempts, allocated>>20)
}

func BenchmarkSolveAttempts(b *testing.B) {
	puzzle := unsolvablePuzzle(8 * 1024)
	b.ReportAllocs()
	for b.Loop() {
		_, _ = solve(context.Background(), puzzle, 10)
	}
}
