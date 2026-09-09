package walletconnect

import "strings"

// Why a handshake was rejected, and whether repeating it could help.
const (
	dialFailureClockSkew   = "clock_skew"
	dialFailureAuth        = "auth"
	dialFailureProject     = "project"
	dialFailureRateLimited = "rate_limited"
	dialFailureServer      = "server"
	dialFailureProxy       = "proxy"
	dialFailureNetwork     = "network"
	dialFailureUnknown     = "unknown"
)

type dialFailure struct {
	class     string
	retryable bool
}

// status is 0 when the relay never answered.
func classifyDialFailure(status int, body string) dialFailure {
	switch {
	case status == 0:
		return dialFailure{class: dialFailureNetwork, retryable: true}
	case status == 401:
		// The relay names both skew directions in the body.
		if strings.Contains(body, "not yet valid") || strings.Contains(body, "is expired") {
			return dialFailure{class: dialFailureClockSkew}
		}
		return dialFailure{class: dialFailureAuth}
	case status == 403:
		return dialFailure{class: dialFailureProject}
	case status == 429:
		return dialFailure{class: dialFailureRateLimited, retryable: true}
	case status == 407 || status == 502 || status == 504:
		// The relay answers JSON; anything else came from a proxy.
		if !strings.HasPrefix(strings.TrimSpace(body), "{") {
			return dialFailure{class: dialFailureProxy, retryable: true}
		}
		return dialFailure{class: dialFailureServer, retryable: true}
	case status >= 500:
		return dialFailure{class: dialFailureServer, retryable: true}
	}
	return dialFailure{class: dialFailureUnknown}
}
