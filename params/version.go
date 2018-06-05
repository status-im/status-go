package params

import (
	"fmt"
)

const (
	// VersionMajor is a major version component of the current release
	VersionMajor = 0

	// VersionMinor is a minor version component of the current release
	VersionMinor = 9

	// VersionPatch is a patch version component of the current release
	VersionPatch = 9
)

// VersionMeta is metadata to append to the version string
var VersionMeta string // rely on linker: -ldflags -X github.com/status-im/status-go/params.VersionMeta"

// Version exposes string representation of program version.
var Version = fmt.Sprintf("%d.%d.%d-%s", VersionMajor, VersionMinor, VersionPatch, VersionMeta)
