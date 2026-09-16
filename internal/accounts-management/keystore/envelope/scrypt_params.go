//go:build !test_fast_kdf

package envelope

// KEK derivation cost. scrypt allocates a 128*N*r scratch buffer per derivation, so N caps the
// transient memory spike: 1<<16 with r=8 keeps it at 64 MiB, chosen to reduce memory pressure on
// mobile (1<<18 caused 256 MiB spikes). This KDF is the primary password-hardening barrier
// protecting the DEK, which is why it is deliberately far above the LightScrypt cost used for
// keystore files. The parameters are stored in the wrapped-DEK file, so a file written with any
// cost stays readable.
const (
	scryptN = 1 << 16
	scryptP = 1
)
