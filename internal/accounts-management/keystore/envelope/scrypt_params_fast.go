//go:build test_fast_kdf

package envelope

// Minimum KEK derivation cost for unit test builds, matching go-ethereum's own
// veryLightScryptN. Never set this tag for a shipped binary.
const (
	scryptN = 2
	scryptP = 1
)
