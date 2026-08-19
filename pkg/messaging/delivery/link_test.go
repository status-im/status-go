package delivery

import (
	"testing"

	"github.com/stretchr/testify/require"

	deliverymsg "github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"
)

// TestLinksAgainstLiblogosdelivery is a build/link smoke test: it creates a node
// through the Messaging API and releases it again, without starting it. It
// fails to compile or link if liblogosdelivery is missing or its C ABI has
// drifted from the bindings, which is the whole point of it existing.
func TestLinksAgainstLiblogosdelivery(t *testing.T) {
	client, err := deliverymsg.New(deliverymsg.Config{
		Mode:   deliverymsg.ModeEdge,
		Preset: deliverymsg.PresetStatusProd,
	})
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, client.Close())
}
