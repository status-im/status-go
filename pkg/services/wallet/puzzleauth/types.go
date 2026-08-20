package puzzleauth

import (
	"time"
)

// Argon2Params represents Argon2id algorithm parameters
type Argon2Params struct {
	MemoryKB int `json:"memory_kb"`
	Time     int `json:"time"`
	Threads  int `json:"threads"`
	KeyLen   int `json:"key_len"`
}

// Puzzle represents a proof-of-work puzzle from the server
type Puzzle struct {
	Challenge    string       `json:"challenge"`
	Salt         string       `json:"salt"`
	Difficulty   int          `json:"difficulty"`
	ExpiresAt    string       `json:"expires_at"`
	HMAC         string       `json:"hmac"`
	Argon2Params Argon2Params `json:"argon2_params"`
}

// Solution represents a solved puzzle
type Solution struct {
	Challenge string `json:"challenge"`
	Salt      string `json:"salt"`
	Nonce     uint64 `json:"nonce"`
	ArgonHash string `json:"argon_hash"`
	HMAC      string `json:"hmac"`
	ExpiresAt string `json:"expires_at"`
}

// TokenResponse represents the response from the auth server
type TokenResponse struct {
	Token        string `json:"token"`
	ExpiresAt    string `json:"expires_at"`
	RequestLimit int    `json:"request_limit"`
}

// TokenData represents cached token data
type TokenData struct {
	Token     string
	ExpiresAt time.Time
}

// IsValid checks if the token is still valid
func (t *TokenData) IsValid() bool {
	if t == nil || t.Token == "" {
		return false
	}
	// Consider token expired 5 minutes before actual expiry
	expiryBuffer := 5 * time.Minute
	return time.Now().Before(t.ExpiresAt.Add(-expiryBuffer))
}
