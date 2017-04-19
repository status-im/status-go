package params

import (
	"fmt"
)

const (
	VersionMajor = 0          // Major version component of the current release
	VersionMinor = 9          // Minor version component of the current release
	VersionPatch = 6          // Patch version component of the current release
	VersionMeta  = "unstable" // Version metadata to append to the version string
)

// Version exposes string representation of program version.
var Version = fmt.Sprintf("%d.%d.%d-%s", VersionMajor, VersionMinor, VersionPatch, VersionMeta)
