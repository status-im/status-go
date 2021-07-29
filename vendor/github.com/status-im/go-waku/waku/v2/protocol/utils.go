package protocol

import "strings"

func FulltextMatch(expectedProtocol string) func(string) bool {
	return func(receivedProtocol string) bool {
		return receivedProtocol == expectedProtocol
	}
}

func PrefixTextMatch(prefix string) func(string) bool {
	return func(receivedProtocol string) bool {
		return strings.HasPrefix(receivedProtocol, prefix)
	}
}
