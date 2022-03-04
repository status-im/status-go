package settings

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyncSettingField_MarshalJSON(t *testing.T) {
	cs := []struct{
		Field SyncSettingField
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
