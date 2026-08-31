//go:build test_fast_kdf

package envelope

// Reduced KEK derivation cost for unit test builds. Never set this tag for a shipped binary.
const (
	scryptN = 1 << 12
	scryptP = 1
)
