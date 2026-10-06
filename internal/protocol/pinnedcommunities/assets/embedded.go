package assets

import _ "embed" // for go:embed

//go:embed 0x02b5bdaf5a25fcfe2ee14c501fab1836b8de57f61621080c3d52073d16de0d98d6.rawpayload
var statusCommunity []byte

// Payloads maps community IDs to raw ApplicationMetadataMessage bytes shipped in the binary.
// The slices live in the binary's data section, not the heap: treat them as read-only.
var Payloads = map[string][]byte{
	"0x02b5bdaf5a25fcfe2ee14c501fab1836b8de57f61621080c3d52073d16de0d98d6": statusCommunity,
}
