package puzzleauth

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckDifficulty(t *testing.T) {
	tests := []struct {
		name       string
		hash       string
		difficulty int
		expected   bool
	}{
		{
			name:       "zero difficulty always passes",
			hash:       "abcdef123456",
			difficulty: 0,
			expected:   true,
		},
		{
			name:       "one leading zero passes",
			hash:       "0abcdef123456",
			difficulty: 1,
			expected:   true,
		},
		{
			name:       "two leading zeros passes",
			hash:       "00abcdef123456",
			difficulty: 2,
			expected:   true,
		},
		{
			name:       "no leading zeros fails",
			hash:       "abcdef123456",
			difficulty: 1,
			expected:   false,
		},
		{
			name:       "one leading zero fails for difficulty 2",
			hash:       "0abcdef123456",
			difficulty: 2,
			expected:   false,
		},
		{
			name:       "hash too short",
			hash:       "00",
			difficulty: 3,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkDifficulty(tt.hash, tt.difficulty)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestSolve(t *testing.T) {
	tests := []struct {
		name        string
		puzzle      *Puzzle
		expectError bool
	}{
		{
			name:        "nil puzzle returns error",
			puzzle:      nil,
			expectError: true,
		},
		{
			name: "invalid salt hex returns error",
			puzzle: &Puzzle{
				Challenge:  "test",
				Salt:       "invalid-hex",
				Difficulty: 1,
				Argon2Params: Argon2Params{
					MemoryKB: 64,
					Time:     1,
					Threads:  1,
					KeyLen:   32,
				},
			},
			expectError: true,
		},
		{
			name: "valid puzzle with low difficulty should solve",
			puzzle: &Puzzle{
				Challenge:  "testchallenge123",
				Salt:       hex.EncodeToString([]byte("testsalt12345678")),
				Difficulty: 1,
				HMAC:       "testhmac",
				ExpiresAt:  "2025-12-31T23:59:59Z",
				Argon2Params: Argon2Params{
					MemoryKB: 64,
					Time:     1,
					Threads:  1,
					KeyLen:   32,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			solution, err := Solve(tt.puzzle)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, solution)
			} else {
				require.NoError(t, err)
				require.NotNil(t, solution)
				require.Equal(t, tt.puzzle.Challenge, solution.Challenge)
				require.Equal(t, tt.puzzle.Salt, solution.Salt)
				require.Equal(t, tt.puzzle.HMAC, solution.HMAC)
				require.Equal(t, tt.puzzle.ExpiresAt, solution.ExpiresAt)
				require.NotEmpty(t, solution.ArgonHash)

				// Verify the solution has the required difficulty
				require.True(t, checkDifficulty(solution.ArgonHash, tt.puzzle.Difficulty),
					"solution hash %s does not meet difficulty %d", solution.ArgonHash, tt.puzzle.Difficulty)
			}
		})
	}
}

func TestSolve_HighDifficulty(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high difficulty test in short mode")
	}

	// This test verifies that solve can handle moderate difficulty
	puzzle := &Puzzle{
		Challenge:  "challenge",
		Salt:       hex.EncodeToString([]byte("salt12345678")),
		Difficulty: 2,
		HMAC:       "hmac",
		ExpiresAt:  "2025-12-31T23:59:59Z",
		Argon2Params: Argon2Params{
			MemoryKB: 64,
			Time:     1,
			Threads:  1,
			KeyLen:   32,
		},
	}

	solution, err := Solve(puzzle)
	require.NoError(t, err)
	require.NotNil(t, solution)
	require.True(t, checkDifficulty(solution.ArgonHash, puzzle.Difficulty))
}
