package api

import (
	"fmt"
	"strings"
)

// AppState represents if the app is in foreground, background or some other state
type AppState string

func (a AppState) String() string {
	return string(a)
}

// Specific app states
// see https://facebook.github.io/react-native/docs/appstate.html
const (
	AppStateForeground = AppState("active") // these constant values are kept in sync with React Native
	AppStateBackground = AppState("background")
	AppStateInactive   = AppState("inactive")

	AppStateInvalid = AppState("")
)

// validAppStates returns an immutable set of valid states.
func validAppStates() []AppState {
	return []AppState{AppStateInactive, AppStateBackground, AppStateForeground}
}

func (a AppState) IsValid() bool {
	for _, state := range validAppStates() {
		if a == state {
			return true
		}
	}
	return false
}

// ParseAppState creates AppState from a string
func ParseAppState(stateString string) (AppState, error) {
	// a bit of cleaning up
	stateString = strings.ToLower(strings.TrimSpace(stateString))

	for _, state := range validAppStates() {
		if stateString == state.String() {
			return state, nil
		}
	}

	return AppStateInvalid, fmt.Errorf("could not parse app state: %s", stateString)
}
