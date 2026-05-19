// Package validate provides ENS name validation predicates.
//
// Pre-ENSv2 code in this repo restricted lookups to *.eth suffixes via
// regular expressions. ENSv2 broadens the resolution surface to DNS imports
// (e.g. alice.xyz) and arbitrary subnames, so any code path that accepts
// user-typed names must use IsLikelyENSName rather than a strict suffix
// check. The strict .eth check should be reserved for the Status username
// registrar flow.
package validate

import "strings"

// IsLikelyENSName implements the heuristic recommended in the ENSv2
// readiness guide: "contains a dot and is longer than two characters".
// This intentionally accepts non-.eth names so DNS imports and L2 subnames
// resolve correctly.
func IsLikelyENSName(name string) bool {
	return len(name) > 2 && strings.Contains(name, ".")
}
