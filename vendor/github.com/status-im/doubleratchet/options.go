package doubleratchet

import "fmt"

// option is a constructor option.
type option func(*state) error

// WithMaxSkip specifies the maximum number of skipped message in a single chain.
func WithMaxSkip(n int) option {
	return func(s *state) error {
		if n < 0 {
			return fmt.Errorf("n must be non-negative")
		}
		s.MaxSkip = uint(n)
		return nil
	}
}

// WithMaxKeep specifies the maximum number of ratchet steps before a message is deleted.
func WithMaxKeep(n int) option {
	return func(s *state) error {
		if n < 0 {
			return fmt.Errorf("n must be non-negative")
		}
		s.MaxKeep = uint(n)
		return nil
	}
}

// WithKeysStorage replaces the default keys storage with the specified.
func WithKeysStorage(ks KeysStorage) option {
	return func(s *state) error {
		if ks == nil {
			return fmt.Errorf("KeysStorage mustn't be nil")
		}
		s.MkSkipped = ks
		return nil
	}
}

// WithCrypto replaces the default cryptographic supplement with the specified.
func WithCrypto(c Crypto) option {
	return func(s *state) error {
		if c == nil {
			return fmt.Errorf("Crypto mustn't be nil")
		}
		s.Crypto = c
		return nil
	}
}
