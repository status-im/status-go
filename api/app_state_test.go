package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAppType(t *testing.T) {
	check := func(input string, expectedState AppState, expectError bool) {
		actualState, err := ParseAppState(input)
		assert.Equalf(t, expectedState, actualState, "unexpected result from parseAppState")
		if expectError {
			assert.NotNil(t, err, "error should not be nil")
		}
	}

	check("active", AppStateForeground, false)
	check("background", AppStateBackground, false)
	check("inactive", AppStateInactive, false)
	check(" acTIVE ", AppStateForeground, false)
	check("    backGROUND  ", AppStateBackground, false)
	check("   INACTIVE   ", AppStateInactive, false)
	check("", AppStateInvalid, true)
	check("back ground", AppStateInvalid, true)
	check(" back ground ", AppStateInvalid, true)
	check("      ", AppStateInvalid, true)
}
