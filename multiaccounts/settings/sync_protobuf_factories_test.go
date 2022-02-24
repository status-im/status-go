package settings

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/protobuf"
)

type testCriteria struct {
	Name     string
	Input    inputCriteria
	Expected expectedCriteria
}

type inputCriteria struct {
	Value interface{}
	Clock uint64
}

type expectedCriteria struct {
	Value int64
	Clock uint64
	AMT   protobuf.ApplicationMetadataMessage_Type
}

func TestProfilePicturesVisibilityProtobufFactory(t *testing.T) {
	var v interface{}
	err := json.Unmarshal([]byte(`3`), &v)
	require.NoError(t, err)

	expected := expectedCriteria{3, 123, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PROFILE_PICTURES_VISIBILITY}

	cs := []testCriteria{
		{"json.Unmarshal int into interface{}", inputCriteria{Value: v, Clock: 123}, expected},
		{"ProfilePicturesVisibilityType", inputCriteria{Value: ProfilePicturesVisibilityNone, Clock: 123}, expected},
		{"int64", inputCriteria{Value: int64(3), Clock: 123}, expected},
	}

	for _, c := range cs {
		a := require.New(t)

		msg, amt, err := profilePicturesVisibilityProtobufFactory(c.Input.Value, c.Input.Clock)
		a.NoError(err, c.Name)

		ppvp, ok := msg.(*protobuf.SyncSettingProfilePicturesVisibility)
		a.True(ok, c.Name)
		a.Equal(c.Expected.Value, ppvp.Value, c.Name)
		a.Equal(c.Expected.Clock, ppvp.Clock, c.Name)
		a.Equal(c.Expected.AMT, amt, c.Name)
	}
}
