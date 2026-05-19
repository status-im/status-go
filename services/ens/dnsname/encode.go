// Package dnsname encodes ENS names into DNS wire format.
//
// The ENSv2 Universal Resolver takes names in DNS wire format, not as
// dot-separated strings. The encoding is a sequence of length-prefixed
// labels followed by a zero byte:
//
//	"alice.eth" -> 0x05 "alice" 0x03 "eth" 0x00
package dnsname

import (
	"errors"
	"strings"
)

const maxLabelLen = 63 // RFC 1035

var (
	errEmptyLabel   = errors.New("dnsname: empty label")
	errLabelTooLong = errors.New("dnsname: label exceeds 63 bytes")
)

// Encode returns the DNS wire-format encoding of name. An empty string
// encodes to a single zero byte (the DNS root).
func Encode(name string) ([]byte, error) {
	if name == "" {
		return []byte{0x00}, nil
	}
	labels := strings.Split(name, ".")
	size := 1 // trailing zero
	for _, l := range labels {
		size += 1 + len(l)
	}
	out := make([]byte, 0, size)
	for _, l := range labels {
		if len(l) == 0 {
			return nil, errEmptyLabel
		}
		if len(l) > maxLabelLen {
			return nil, errLabelTooLong
		}
		out = append(out, byte(len(l)))
		out = append(out, l...)
	}
	out = append(out, 0x00)
	return out, nil
}
