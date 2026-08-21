package assets

import "embed"

// FS contains pinned community payloads bundled into the binary.
//
//go:embed *.rawpayload.hex
var FS embed.FS
