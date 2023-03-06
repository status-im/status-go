package settings

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/protobuf"
)

func TestSyncSettingField_MarshalJSON(t *testing.T) {
	cs := []struct {
		Field    SyncSettingField
		Expected []byte
	}{
		{
			Field: SyncSettingField{
				Currency,
				"eth",
			},
			Expected: []byte("{\"name\":\"currency\",\"value\":\"eth\"}"),
		},
		{
			Field: SyncSettingField{
				ProfilePicturesShowTo,
				ProfilePicturesShowToNone,
			},
			Expected: []byte("{\"name\":\"profile-pictures-show-to\",\"value\":3}"),
		},
		{
			Field: SyncSettingField{
				MessagesFromContactsOnly,
				false,
			},
			Expected: []byte("{\"name\":\"messages-from-contacts-only\",\"value\":false}"),
		},
	}

	for _, c := range cs {
		js, err := json.Marshal(c.Field)
		require.NoError(t, err)
		require.Equal(t, c.Expected, js)
	}
}

func TestJSONEncoding(t *testing.T) {
	settings := Settings{
		PublicKey: "0x04deaafa03e3a646e54a36ec3f6968c1d3686847d88420f00c0ab6ee517ee1893398fca28aacd2af74f2654738c21d10bad3d88dc64201ebe0de5cf1e313970d3d",
	}
	encoded, err := json.Marshal(settings)
	require.NoError(t, err)

	require.True(t, strings.Contains(string(encoded), "\"compressedKey\":\"zQ3shudJrBctPznsRLvbsCtvZFTdi3b34uzYDuqE9Wq9m9T1C\""))
	require.True(t, strings.Contains(string(encoded), "\"emojiHash\""))
}

// TestGetFieldFromProtobufType checks if all the protobuf.SyncSetting_Type_value are assigned to a SettingField
func TestGetFieldFromProtobufType(t *testing.T) {
	for _, sst := range protobuf.SyncSetting_Type_value {
		_, err := GetFieldFromProtobufType(protobuf.SyncSetting_Type(sst))
		if sst == 0 {
			require.Error(t, err, "do not have a SettingField for the unknown type")
			continue
		}
		if err != nil {
			require.NoError(t, err)
		}
	}
}
