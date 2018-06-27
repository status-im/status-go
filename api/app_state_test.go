package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAppType(t *testing.T) {
	check := func(input string, expectedState appState, expectError bool) {
		actualState, err := parseAppState(input)
		assert.Equalf(t, expectedState, actualState, "unexpected result from parseAppState")
		if expectError {
			assert.NotNil(t, err, "error should not be nil")
		}
	}

	check("active", appStateForeground, false)
	check("background", appStateBackground, false)
	check("inactive", appStateInactive, false)
	check(" acTIVE ", appStateForeground, false)
	check("    backGROUND  ", appStateBackground, false)
	check("   INACTIVE   ", appStateInactive, false)
	check("", appStateInvalid, true)
	check("back ground", appStateInvalid, true)
	check(" back ground ", appStateInvalid, true)
	check("      ", appStateInvalid, true)
}
