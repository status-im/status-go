package puzzleauth

import (
	"context"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const maxAttempts = 1000000

// Solve attempts to solve the puzzle by finding a nonce that produces
// a hash with the required number of leading zeros
func Solve(ctx context.Context, puzzle *Puzzle) (*Solution, error) {
	if puzzle == nil {
		return nil, fmt.Errorf("puzzle cannot be nil")
	}

	if puzzle.Difficulty < 0 {
		return nil, fmt.Errorf("invalid difficulty: %d", puzzle.Difficulty)
	}
	if puzzle.Argon2Params.MemoryKB <= 0 || puzzle.Argon2Params.MemoryKB > 4*1024*1024 { // max 4GB
		return nil, fmt.Errorf("invalid memory_kb: %d", puzzle.Argon2Params.MemoryKB)
	}
	if puzzle.Argon2Params.Time <= 0 || puzzle.Argon2Params.Time > 100 {
		return nil, fmt.Errorf("invalid time: %d", puzzle.Argon2Params.Time)
	}
	if puzzle.Argon2Params.Threads <= 0 || puzzle.Argon2Params.Threads > 255 {
		return nil, fmt.Errorf("invalid threads: %d", puzzle.Argon2Params.Threads)
	}
	if puzzle.Argon2Params.KeyLen <= 0 || puzzle.Argon2Params.KeyLen > 1024 {
		return nil, fmt.Errorf("invalid key_len: %d", puzzle.Argon2Params.KeyLen)
	}

	saltBytes, err := hex.DecodeString(puzzle.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	memory := uint32(puzzle.Argon2Params.MemoryKB)
	time := uint32(puzzle.Argon2Params.Time)
	threads := uint8(puzzle.Argon2Params.Threads)
	keyLen := uint32(puzzle.Argon2Params.KeyLen)

	// Try different nonces until we find one that meets the difficulty requirement
	for nonce := uint64(0); nonce < maxAttempts; nonce++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Create input: challenge + salt + nonce
		input := fmt.Sprintf("%s%s%d", puzzle.Challenge, puzzle.Salt, nonce)

		// Compute Argon2id hash
		hash := argon2.IDKey([]byte(input), saltBytes, time, memory, threads, keyLen)
		argonHash := hex.EncodeToString(hash)

		// Check if hash meets difficulty requirement
		if checkDifficulty(argonHash, puzzle.Difficulty) {
			return &Solution{
				Challenge: puzzle.Challenge,
				Salt:      puzzle.Salt,
				Nonce:     nonce,
				ArgonHash: argonHash,
				HMAC:      puzzle.HMAC,
				ExpiresAt: puzzle.ExpiresAt,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to solve puzzle within %d attempts", maxAttempts)
}

// checkDifficulty checks if the hash has the required number of leading zeros
func checkDifficulty(hash string, difficulty int) bool {
	if len(hash) < difficulty {
		return false
	}

	for i := 0; i < difficulty; i++ {
		if hash[i] != '0' {
			return false
		}
	}

	return true
}
