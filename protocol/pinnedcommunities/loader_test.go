package pinnedcommunities

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"

	protocolcommon "github.com/status-im/status-go/protocol/common"
	protocolprotobuf "github.com/status-im/status-go/protocol/protobuf"
)

func TestLoadFromDir(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "0xbbb.rawpayload.hex"), []byte("03\n"), 0o600)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "0xaaa.rawpayload.hex"), []byte("0102"), 0o600)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("ignore"), 0o600)
	require.NoError(t, err)

	payloads, err := LoadFromDir(tmpDir)
	require.NoError(t, err)
	require.Len(t, payloads, 2)

	require.Equal(t, "0xaaa", payloads[0].CommunityID)
	require.Equal(t, "0xaaa.rawpayload.hex", payloads[0].FileName)
	require.Equal(t, []byte{0x01, 0x02}, payloads[0].RawPayload)

	require.Equal(t, "0xbbb", payloads[1].CommunityID)
	require.Equal(t, "0xbbb.rawpayload.hex", payloads[1].FileName)
	require.Equal(t, []byte{0x03}, payloads[1].RawPayload)
}

func TestLoadFromDirInvalidHex(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "0xaaa.rawpayload.hex"), []byte("zz"), 0o600)
	require.NoError(t, err)

	_, err = LoadFromDir(tmpDir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode pinned community payload")
}

func TestLoadFromDirEmptyContent(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "0xaaa.rawpayload.hex"), []byte(" \n\t"), 0o600)
	require.NoError(t, err)

	_, err = LoadFromDir(tmpDir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty content")
}

func TestEmbeddedPinnedCommunitiesAreImportable(t *testing.T) {
	payloads, err := LoadEmbedded()
	require.NoError(t, err)
	require.NotEmpty(t, payloads, "expected at least one pinned community payload")

	for _, p := range payloads {
		var metadata protocolprotobuf.ApplicationMetadataMessage
		err := proto.Unmarshal(p.RawPayload, &metadata)
		require.NoError(t, err, "metadata should decode for %s", p.FileName)

		require.Equal(t, protocolprotobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, metadata.Type,
			"metadata type should be COMMUNITY_DESCRIPTION for %s", p.FileName)

		signer, err := protocolcommon.RecoverKey(&metadata)
		require.NoError(t, err, "signer should recover for %s", p.FileName)
		require.NotNil(t, signer, "signer should not be nil for %s", p.FileName)

		var description protocolprotobuf.CommunityDescription
		err = proto.Unmarshal(metadata.Payload, &description)
		require.NoError(t, err, "community description should decode for %s", p.FileName)

		if description.ID != "" {
			require.True(t, strings.EqualFold(description.ID, p.CommunityID),
				"community id in filename (%s) should match payload id (%s) for %s",
				p.CommunityID, description.ID, p.FileName)
		}
	}
}
