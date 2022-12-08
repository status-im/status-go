package protocol

import "strings"

// FulltextMatch is the default matching function used for checking if a peer
// supports a protocol or not
func FulltextMatch(expectedProtocol string) func(string) bool {
	return func(receivedProtocol string) bool {
		return receivedProtocol == expectedProtocol
	}
}

// PrefixTextMatch is a matching function used for checking if a peer's
// supported protocols begin with a particular prefix
func PrefixTextMatch(prefix string) func(string) bool {
	return func(receivedProtocol string) bool {
		return strings.HasPrefix(receivedProtocol, prefix)
	}
}
