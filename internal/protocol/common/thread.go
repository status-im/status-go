package common

import (
	"strings"

	"github.com/rivo/uniseg"
)

const MaxThreadNameLength = 50

// NormalizeThreadName collapses whitespace and limits names to the supported
// grapheme length when truncate is true. Its second result reports overflow.
func NormalizeThreadName(value string, truncate bool) (string, bool) {
	name := strings.Join(strings.Fields(value), " ")
	if name == "" {
		return "", false
	}

	graphemes := uniseg.NewGraphemes(name)
	for count := 0; graphemes.Next(); count++ {
		if count == MaxThreadNameLength {
			if !truncate {
				return name, true
			}
			start, _ := graphemes.Positions()
			return name[:start], true
		}
	}

	return name, false
}
