package assets

import (
	_ "embed" // for go:embed
	"sort"
)

//go:embed 0x02b5bdaf5a25fcfe2ee14c501fab1836b8de57f61621080c3d52073d16de0d98d6.rawpayload
var statusCommunity []byte

// payloads maps community IDs to raw ApplicationMetadataMessage bytes shipped in the binary.
// The slices live in the binary's data section, not the heap.
var payloads = map[string][]byte{
	"0x02b5bdaf5a25fcfe2ee14c501fab1836b8de57f61621080c3d52073d16de0d98d6": statusCommunity,
}

// IDs returns the IDs of the embedded communities, sorted.
func IDs() []string {
	ids := make([]string, 0, len(payloads))
	for id := range payloads {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Payload returns the embedded payload of a community. It aliases the binary's data and must not be modified.
func Payload(id string) ([]byte, bool) {
	p, ok := payloads[id]
	return p, ok
}
