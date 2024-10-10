package requests

import (
	"testing"

	"github.com/stretchr/testify/require"

	protocolRequests "github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/server/pairing"
)

func TestInputConnectionStringForBootstrapping_Validate(t *testing.T) {
	t.Run("Valid input", func(t *testing.T) {
		input := &InputConnectionStringForBootstrapping{
			ConnectionString: "some-connection-string",
			ReceiverClientConfig: &pairing.ReceiverClientConfig{
				ReceiverConfig: &pairing.ReceiverConfig{
					CreateAccount: &protocolRequests.CreateAccount{},
				},
			},
		}

		err := input.Validate()
		require.NoError(t, err)
	})

	t.Run("Missing ReceiverClientConfig", func(t *testing.T) {
		input := &InputConnectionStringForBootstrapping{
			ConnectionString:     "some-connection-string",
			ReceiverClientConfig: nil,
		}

		err := input.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "ReceiverClientConfig")
		require.Contains(t, err.Error(), "required")
	})
}
