package thirdparty

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/services/wallet/puzzleauth"
)

var ErrPuzzleAuthClientRequired = errors.New("PuzzleAuthClient is required for AuthTypePOW")

// AuthType specifies how requests are authenticated.
type AuthType int

const (
	// AuthTypeAPIKey means API key is embedded in URL (direct Alchemy), no auth headers.
	AuthTypeAPIKey AuthType = iota
	// AuthTypeBasic means Basic auth (user/password) in headers.
	AuthTypeBasic
	// AuthTypePOW means proof-of-work auth via puzzleauth.Client.
	AuthTypePOW
	// AuthTypeNone means no authentication (e.g. proxy without creds or puzzle).
	AuthTypeNone
)

// AuthParams holds authentication configuration for AuthTransport.
type AuthParams struct {
	Type             AuthType
	APIKey           security.SensitiveString
	Creds            *BasicCreds
	PuzzleAuthClient *puzzleauth.Client
}

// AuthTransport executes HTTP requests with auth headers and retry logic.
type AuthTransport struct {
	httpClient *http.Client
	auth       AuthParams
	providerID string
}

// NewAuthTransport creates a new AuthTransport.
func NewAuthTransport(httpClient *http.Client, auth AuthParams, providerID string) *AuthTransport {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: time.Minute}
	}
	return &AuthTransport{
		httpClient: httpClient,
		auth:       auth,
		providerID: providerID,
	}
}

// Do executes the request with auth applied and retries as appropriate.
func (t *AuthTransport) Do(req *http.Request) (*http.Response, error) {
	t.applyAuth(req)
	return t.doWithRetries(req)
}

func (t *AuthTransport) authTypeName() string {
	switch t.auth.Type {
	case AuthTypeAPIKey:
		return "APIKey"
	case AuthTypeBasic:
		return "Basic"
	case AuthTypePOW:
		return "POW"
	case AuthTypeNone:
		return "None"
	default:
		return fmt.Sprintf("Unknown(%d)", t.auth.Type)
	}
}

// Auth returns the auth params (e.g. for reading APIKey).
func (t *AuthTransport) Auth() AuthParams {
	return t.auth
}

func (t *AuthTransport) applyAuth(req *http.Request) {
	if t.auth.Type == AuthTypeBasic && t.auth.Creds != nil {
		req.SetBasicAuth(t.auth.Creds.User.Reveal(), t.auth.Creds.Password.Reveal())
	}
	// AuthTypeAPIKey: key is in URL, no header needed
	// AuthTypePOW: puzzleauth.Client handles auth internally in DoRequest
	// AuthTypeNone: no auth headers
}

func (t *AuthTransport) doWithRetries(req *http.Request) (*http.Response, error) {
	if t.auth.Type == AuthTypePOW {
		if t.auth.PuzzleAuthClient == nil {
			return nil, ErrPuzzleAuthClientRequired
		}
		return t.auth.PuzzleAuthClient.DoRequest(req)
	}
	return DoWithExponentialBackoff(t.httpClient, req, t.providerID)
}
