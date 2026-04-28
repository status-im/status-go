package thirdparty

import (
	"fmt"
	"net/http"
	"time"

	"github.com/status-im/status-go/pkg/security"
)

// AuthType specifies how requests are authenticated.
type AuthType int

const (
	// AuthTypeAPIKey means API key is embedded in URL (direct Alchemy), no auth headers.
	AuthTypeAPIKey AuthType = iota
	// AuthTypeBasic means Basic auth (user/password) in headers.
	AuthTypeBasic
	// AuthTypeNone means no custom auth headers (e.g. proxy without creds, or puzzle via custom [http.Client].Transport).
	AuthTypeNone
)

// AuthParams holds authentication configuration for AuthTransport.
type AuthParams struct {
	Type   AuthType
	APIKey security.SensitiveString
	Creds  *BasicCreds
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
	// AuthTypeNone: no auth headers; optional Transport on httpClient (e.g. puzzle) may add them
}

func (t *AuthTransport) doWithRetries(req *http.Request) (*http.Response, error) {
	return DoWithExponentialBackoff(t.httpClient, req, t.providerID)
}
