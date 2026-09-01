//go:build !test_fast_kdf

package envelope

// KEK derivation cost, matching go-ethereum's StandardScryptN/P. The parameters are
// stored in the wrapped-DEK file, so a file written with any cost stays readable.
const (
	scryptN = 1 << 18
	scryptP = 1
)
