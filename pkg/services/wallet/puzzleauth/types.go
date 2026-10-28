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

const (
	// maxExpiryBuffer caps how early a token is treated as expired.
	maxExpiryBuffer = time.Minute
	// maxRefreshLead caps how long before that point a request starts a background refresh.
	maxRefreshLead = 3 * time.Minute
)

// TokenData represents cached token data
type TokenData struct {
	Token     string
	IssuedAt  time.Time
	ExpiresAt time.Time
	// RequestLimit is how many requests the server accepts with the token; 0 means no limit.
	RequestLimit int
	// Used counts the requests sent with the token.
	Used int
}

// IsValid reports whether the token can still be sent: it is treated as expired
// a tenth of its lifetime (at most maxExpiryBuffer) before ExpiresAt, or once
// RequestLimit requests were sent with it.
func (t *TokenData) IsValid() bool {
	return t.validAt(time.Now())
}

func (t *TokenData) validAt(now time.Time) bool {
	if t == nil || t.Token == "" {
		return false
	}
	if t.limitReached() {
		return false
	}
	return now.Before(t.validUntil())
}

// refreshDueAt reports whether at most a quarter of the token lifetime
// (at most maxRefreshLead) is left before it stops being sent.
func (t *TokenData) refreshDueAt(now time.Time) bool {
	return !now.Before(t.validUntil().Add(-t.lifetimeShare(4, maxRefreshLead)))
}

func (t *TokenData) validUntil() time.Time {
	return t.ExpiresAt.Add(-t.lifetimeShare(10, maxExpiryBuffer))
}

// lifetimeShare returns 1/parts of the token lifetime, at most limit.
func (t *TokenData) lifetimeShare(parts time.Duration, limit time.Duration) time.Duration {
	if t.IssuedAt.IsZero() {
		return limit
	}
	return min(limit, t.ExpiresAt.Sub(t.IssuedAt)/parts)
}

func (t *TokenData) limitReached() bool {
	return t.RequestLimit > 0 && t.Used >= t.RequestLimit
}
