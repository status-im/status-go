package protocol

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/protobuf"
)

const expectedJPEG = "data:image/jpeg;base64,/9j/2wBDAAMCAgICAgMCAgIDAwMDBAYEBAQEBAgGBgUGCQgKCgkICQkKDA8MCgsOCwkJDRENDg8QEBEQCgwSExIQEw8QEBD/yQALCAABAAEBAREA/8wABgAQEAX/2gAIAQEAAD8A0s8g/9k="

func TestPrepareContentImage(t *testing.T) {
	file, err := os.Open("../_assets/tests/test.jpg")
	require.NoError(t, err)
	defer file.Close()

	payload, err := ioutil.ReadAll(file)
	require.NoError(t, err)

	message := &Message{}
	message.ContentType = protobuf.ChatMessage_IMAGE
	image := protobuf.ImageMessage{
		Payload: payload,
		Type:    protobuf.ImageMessage_JPEG,
	}
	message.Payload = &protobuf.ChatMessage_Image{Image: &image}

	require.NoError(t, message.PrepareContent())
	require.Equal(t, message.Base64Image, expectedJPEG)
}
