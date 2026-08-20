package walletconnect

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const ClientIDValue = "walletconnect"

var (
	ErrInvalidURI = errors.New("invalid WalletConnect URI")
	topicRe       = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
)

// ParseURI parses a WalletConnect v2 URI string.
// Format: wc:<topic>@<version>?relay-protocol=irn&symKey=<hex>&projectId=<uuid>
func ParseURI(uri string) (*ParsedURI, error) {
	uri = strings.TrimSpace(uri)

	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "wc" {
		return nil, ErrInvalidURI
	}

	// Opaque = "topic@version"
	atIdx := strings.LastIndex(u.Opaque, "@")
	if atIdx < 0 {
		return nil, ErrInvalidURI
	}
	topic := u.Opaque[:atIdx]
	version := u.Opaque[atIdx+1:]

	// Topic must be 64 hex chars (32 bytes)
	if !topicRe.MatchString(topic) {
		return nil, ErrInvalidURI
	}

	q := u.Query()

	relayProtocol := q.Get("relay-protocol")
	if relayProtocol == "" {
		relayProtocol = "irn"
	}

	symKey := q.Get("symKey")
	if symKey == "" {
		return nil, ErrInvalidURI
	}

	parsed := &ParsedURI{
		Topic:         topic,
		Version:       version,
		SymKey:        symKey,
		RelayProtocol: relayProtocol,
		ProjectID:     q.Get("projectId"),
	}

	// Parse expiryTimestamp if present
	if expiryStr := q.Get("expiryTimestamp"); expiryStr != "" {
		if ts, err := strconv.ParseInt(expiryStr, 10, 64); err == nil {
			parsed.ExpiryTimestamp = ts
		}
	}

	return parsed, nil
}

type ParsedURI struct {
	Topic           string
	Version         string
	SymKey          string
	RelayProtocol   string
	ProjectID       string
	ExpiryTimestamp int64
}
