package crypto

import (
	"errors"
)

// ErrRsaKeyTooSmall is returned when trying to generate or parse an RSA key
// that's smaller than 512 bits. Keys need to be larger enough to sign a 256bit
// hash so this is a reasonable absolute minimum.
var ErrRsaKeyTooSmall = errors.New("rsa keys must be >= 512 bits to be useful")
