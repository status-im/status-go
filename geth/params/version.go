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

	// VersionMeta is metadata to append to the version string
	VersionMeta = "unstable"
)

// Version exposes string representation of program version.
var Version = buildVersionString(VersionMajor, VersionMinor, VersionPatch, VersionMeta)

// buildVersionString builds string representation of program version.
func buildVersionString(major, minor, patch int, meta string) string {
	var version string

	if len(meta) > 0 {
		version = fmt.Sprintf("%d.%d.%d-%s", major, minor, patch, meta)
	} else {
		version = fmt.Sprintf("%d.%d.%d", major, minor, patch)
	}

	return version
}
