package settings

import (
	"encoding/json"
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
