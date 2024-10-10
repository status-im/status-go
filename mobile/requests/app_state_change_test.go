package requests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/api"
)

func TestAppStateChange(t *testing.T) {
	t.Run("Valid States", func(t *testing.T) {
		testCases := []api.AppState{
			api.AppStateBackground,
			api.AppStateForeground,
			api.AppStateInactive,
		}

		for _, state := range testCases {
			req := AppStateChange{State: state}
			err := req.Validate()
			require.NoError(t, err, "validation should pass for state: %s", state)
		}
	})

	t.Run("Invalid State", func(t *testing.T) {
		invalidStates := []api.AppState{"invalid-state", api.AppStateInvalid}

		for _, state := range invalidStates {
			req := AppStateChange{State: state}
			err := req.Validate()
			require.Error(t, err, "validation should fail for invalid state: %s", state)
		}
	})
}
