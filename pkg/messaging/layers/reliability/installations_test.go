package reliability

import (
	"testing"

	"github.com/golang/protobuf/proto"
	mvdsproto "github.com/status-im/mvds/protobuf"
	"github.com/stretchr/testify/require"
)

func TestInstallationPayloadSeparatesLegacyAndScopedACKs(t *testing.T) {
	r := newTestReliability(t)
	require.NoError(t, r.Start(noopDispatch))
	t.Cleanup(r.Close)

	first, err := r.QueueInstallationMessage("sender", "first", []byte("sync"))
	require.NoError(t, err)
	second, err := r.QueueInstallationMessage("sender", "second", []byte("sync"))
	require.NoError(t, err)
	legacyID, err := r.QueueLegacySelfMessage([]byte("sync"))
	require.NoError(t, err)
	require.NotEqual(t, first, second)
	require.NotEqual(t, first, legacyID)

	encoded, err := proto.Marshal(&mvdsproto.Payload{Acks: [][]byte{first[:], second[:], legacyID[:]}})
	require.NoError(t, err)
	scoped, legacy, err := r.ProcessInstallationPayload("sender", "first", encoded, true)
	require.NoError(t, err)
	require.Equal(t, [][]byte{first[:]}, scoped.Acks)
	require.Equal(t, [][]byte{legacyID[:]}, legacy.Acks)

	scoped, _, err = r.ProcessInstallationPayload("sender", "second", encoded, false)
	require.NoError(t, err)
	require.Empty(t, scoped.Acks)
}
