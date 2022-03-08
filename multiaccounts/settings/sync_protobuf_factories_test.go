package settings

import (
	"encoding/json"
	"testing"

	"github.com/golang/protobuf/proto"
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

	expected := expectedCriteria{3, 123, protobuf.ApplicationMetadataMessage_SYNC_SETTING}

	cs := []testCriteria{
		{"json.Unmarshal int into interface{}", inputCriteria{Value: v, Clock: 123}, expected},
		{"ProfilePicturesVisibilityType", inputCriteria{Value: ProfilePicturesVisibilityNone, Clock: 123}, expected},
		{"int64", inputCriteria{Value: int64(3), Clock: 123}, expected},
	}

	for _, c := range cs {
		a := require.New(t)

		rm, err := profilePicturesVisibilityProtobufFactory(c.Input.Value, c.Input.Clock, "0x123def")
		a.NoError(err, c.Name)

		ppvp := new(protobuf.SyncSetting)
		err = proto.Unmarshal(rm.Payload, ppvp)
		a.NoError(err, c.Name)

		a.Equal(protobuf.SyncSetting_PROFILE_PICTURES_VISIBILITY, ppvp.Type, c.Name)
		a.Equal(c.Expected.Value, ppvp.GetValueInt64(), c.Name)
		a.Equal(c.Expected.Clock, ppvp.Clock, c.Name)
		a.Equal(c.Expected.AMT, rm.MessageType, c.Name)
	}
}
