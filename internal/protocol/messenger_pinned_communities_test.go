package protocol

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/pinnedcommunities"
	protocolprotobuf "github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/pkg/messaging"
)

func TestBootstrapPinnedCommunitiesImportsIntoDB(t *testing.T) {
	messagingEnv, err := messaging.NewTestMessagingEnvironment()
	require.NoError(t, err)
	require.NoError(t, messagingEnv.Setup(t))

	payloads, err := pinnedcommunities.LoadEmbedded()
	require.NoError(t, err)
	require.NotEmpty(t, payloads, "expected at least one pinned community payload")

	m, err := newRunningTestMessenger(t, messagingEnv, testMessengerConfig{
		extraOptions: []Option{WithEnablePinnedBootstrap(true)},
	})
	require.NoError(t, err)

	waitForPinnedCommunitiesInDB(t, m, payloads)

	for _, p := range payloads {
		community, err := m.FindCommunityInfoFromDB(p.CommunityID)
		require.NoError(t, err, "community should exist in db for %s", p.CommunityID)
		require.NotNil(t, community, "community should be loaded for %s", p.CommunityID)
		require.Equal(t, p.CommunityID, community.IDString(), "imported community id should match file name id")
	}
}

func TestBootstrapPinnedCommunitiesRunsOnlyOncePerProfile(t *testing.T) {
	messagingEnv, err := messaging.NewTestMessagingEnvironment()
	require.NoError(t, err)
	require.NoError(t, messagingEnv.Setup(t))

	payloads, err := pinnedcommunities.LoadEmbedded()
	require.NoError(t, err)
	require.NotEmpty(t, payloads)

	m, err := newRunningTestMessenger(t, messagingEnv, testMessengerConfig{
		extraOptions: []Option{WithEnablePinnedBootstrap(true)},
	})
	require.NoError(t, err)

	waitForPinnedCommunitiesInDB(t, m, payloads)

	// First run imported pinned communities. Now point to a broken dir.
	brokenDir := t.TempDir()
	err = os.WriteFile(filepath.Join(brokenDir, "0xdeadbeef.rawpayload.hex"), []byte("zz"), 0o600)
	require.NoError(t, err)
	t.Setenv("STATUS_GO_PINNED_COMMUNITIES_DIR", brokenDir)

	// Must not fail: second bootstrap should be skipped because profile already has communities.
	err = m.bootstrapPinnedCommunities()
	require.NoError(t, err)

	for _, p := range payloads {
		community, err := m.FindCommunityInfoFromDB(p.CommunityID)
		require.NoError(t, err)
		require.NotNil(t, community)
	}
}

func TestBootstrapPinnedCommunitiesLoadsFromDir(t *testing.T) {
	messagingEnv, err := messaging.NewTestMessagingEnvironment()
	require.NoError(t, err)
	require.NoError(t, messagingEnv.Setup(t))

	payloads, err := pinnedcommunities.LoadEmbedded()
	require.NoError(t, err)
	require.NotEmpty(t, payloads)

	tmpDir := t.TempDir()
	selected := payloads[0]
	err = os.WriteFile(
		filepath.Join(tmpDir, selected.CommunityID+pinnedcommunities.RawPayloadHexSuffix),
		[]byte(hex.EncodeToString(selected.RawPayload)),
		0o600,
	)
	require.NoError(t, err)
	t.Setenv("STATUS_GO_PINNED_COMMUNITIES_DIR", tmpDir)

	m, err := newRunningTestMessenger(t, messagingEnv, testMessengerConfig{})
	require.NoError(t, err)

	err = m.bootstrapPinnedCommunities()
	require.NoError(t, err)

	waitForPinnedCommunitiesInDB(t, m, []pinnedcommunities.Payload{selected})
}

func TestBootstrapPinnedCommunitiesMissingDirSkipsWithoutError(t *testing.T) {
	messagingEnv, err := messaging.NewTestMessagingEnvironment()
	require.NoError(t, err)
	require.NoError(t, messagingEnv.Setup(t))

	t.Setenv("STATUS_GO_PINNED_COMMUNITIES_DIR", filepath.Join(t.TempDir(), "does-not-exist"))

	m, err := newRunningTestMessenger(t, messagingEnv, testMessengerConfig{})
	require.NoError(t, err)

	err = m.bootstrapPinnedCommunities()
	require.NoError(t, err)

	shouldBootstrap, err := m.shouldBootstrapPinnedCommunities()
	require.NoError(t, err)
	require.True(t, shouldBootstrap, "profile should still have zero communities after missing dir")
}

func TestBootstrapPinnedCommunitiesInvalidDirPayloadReturnsError(t *testing.T) {
	messagingEnv, err := messaging.NewTestMessagingEnvironment()
	require.NoError(t, err)
	require.NoError(t, messagingEnv.Setup(t))

	tmpDir := t.TempDir()
	err = os.WriteFile(filepath.Join(tmpDir, "0xdeadbeef.rawpayload.hex"), []byte("zz"), 0o600)
	require.NoError(t, err)
	t.Setenv("STATUS_GO_PINNED_COMMUNITIES_DIR", tmpDir)

	m, err := newRunningTestMessenger(t, messagingEnv, testMessengerConfig{})
	require.NoError(t, err)

	err = m.bootstrapPinnedCommunities()
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode pinned community payload")
}

func TestImportPinnedCommunityPayloadInvalidMetadata(t *testing.T) {
	m := &Messenger{}

	_, err := m.importPinnedCommunityPayload(pinnedcommunities.Payload{
		CommunityID: "0xabc",
		RawPayload:  []byte{0x01, 0x02, 0x03},
		FileName:    "0xabc.rawpayload.hex",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal application metadata")
}

func TestImportPinnedCommunityPayloadUnsupportedType(t *testing.T) {
	m := &Messenger{}

	rawPayload, err := proto.Marshal(&protocolprotobuf.ApplicationMetadataMessage{
		Type:    protocolprotobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
		Payload: []byte{0x01},
	})
	require.NoError(t, err)

	_, err = m.importPinnedCommunityPayload(pinnedcommunities.Payload{
		CommunityID: "0xabc",
		RawPayload:  rawPayload,
		FileName:    "0xabc.rawpayload.hex",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported metadata type")
}

func waitForPinnedCommunitiesInDB(t *testing.T, m *Messenger, payloads []pinnedcommunities.Payload) {
	t.Helper()

	require.Eventually(t, func() bool {
		for _, p := range payloads {
			community, err := m.FindCommunityInfoFromDB(p.CommunityID)
			if err != nil || community == nil || community.IDString() != p.CommunityID {
				return false
			}
		}

		return true
	}, 5*time.Second, 50*time.Millisecond, "pinned communities were not imported into db in time")
}
