package puzzleauth

import (
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const maxAttempts = 1000000

// Solve attempts to solve the puzzle by finding a nonce that produces
// a hash with the required number of leading zeros
func Solve(puzzle *Puzzle) (*Solution, error) {
	if puzzle == nil {
		return nil, fmt.Errorf("puzzle cannot be nil")
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
