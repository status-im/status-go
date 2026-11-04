package pb

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/hash"
	"google.golang.org/protobuf/proto"
)

func TestHash(t *testing.T) {
	expected := []byte{0xa5, 0x91, 0xa6, 0xd4, 0xb, 0xf4, 0x20, 0x40, 0x4a, 0x1, 0x17, 0x33, 0xcf, 0xb7, 0xb1, 0x90, 0xd6, 0x2c, 0x65, 0xbf, 0xb, 0xcd, 0xa3, 0x2b, 0x57, 0xb2, 0x77, 0xd9, 0xad, 0x9f, 0x14, 0x6e}
	result := hash.SHA256([]byte("Hello World"))
	require.Equal(t, expected, result)
}

func TestEnvelopeHash(t *testing.T) {
	msg := new(WakuMessage)
	msg.ContentTopic = "Test"
	msg.Payload = []byte("Hello World")
	msg.Timestamp = proto.Int64(123456789123456789)
	msg.Version = proto.Uint32(1)

	expected := []byte{0xb6, 0x59, 0x60, 0x7f, 0x2a, 0xae, 0x18, 0x84, 0x8d, 0xca, 0xa7, 0xd5, 0x1c, 0xb3, 0x7e, 0x6c, 0xc6, 0xfc, 0x33, 0x40, 0x2c, 0x70, 0x4f, 0xf0, 0xc0, 0x16, 0x33, 0x7d, 0x83, 0xad, 0x61, 0x50}
	result := msg.Hash("test")
	require.Equal(t, ToMessageHash(expected), result)
}

func TestEmptyMeta(t *testing.T) {
	pubsubTopic := "/waku/2/default-waku/proto"
	msg := new(WakuMessage)
	msg.ContentTopic = "/waku/2/default-content/proto"
	msg.Payload = []byte("\x01\x02\x03\x04TEST\x05\x06\x07\x08")

	msg.Meta = []byte{}
	msg.Timestamp = proto.Int64(123456789123456789)
	msg.Version = proto.Uint32(1)

	messageHash := msg.Hash(pubsubTopic)

	require.Equal(t, "0xf0183c2e370e473ff471bbe1028d0d8a940949c02f3007a1ccd21fed356852a0", messageHash.String())
}

func Test13ByteMeta(t *testing.T) {
	pubsubTopic := "/waku/2/default-waku/proto"
	msg := new(WakuMessage)
	msg.ContentTopic = "/waku/2/default-content/proto"
	msg.Payload = []byte("\x01\x02\x03\x04TEST\x05\x06\x07\x08")
	msg.Meta = []byte("\x73\x75\x70\x65\x72\x2d\x73\x65\x63\x72\x65\x74")
	msg.Timestamp = proto.Int64(123456789123456789)
	msg.Version = proto.Uint32(1)

	messageHash := msg.Hash(pubsubTopic)

	require.Equal(t, "0xf673cd2c9c973d685b52ca74c2559e001733a3a31a49ffc7b6e8713decba5a55", messageHash.String())
}

func TestZeroLenPayload(t *testing.T) {
	pubsubTopic := "/waku/2/default-waku/proto"
	msg := new(WakuMessage)
	msg.ContentTopic = "/waku/2/default-content/proto"
	msg.Payload = []byte{}
	msg.Meta = []byte("\x73\x75\x70\x65\x72\x2d\x73\x65\x63\x72\x65\x74")
	msg.Timestamp = proto.Int64(123456789123456789)
	msg.Version = proto.Uint32(1)

	messageHash := msg.Hash(pubsubTopic)

	require.Equal(t, "0x978ccc9a665029f9829d42d84e3a49ad3a4791cce53fb5a8b581ef43ad6b4d2f", messageHash.String())
}

func TestHashWithTimestamp(t *testing.T) {
	pubsubTopic := "/waku/2/default-waku/proto"
	msg := new(WakuMessage)
	msg.ContentTopic = "/waku/2/default-content/proto"
	msg.Payload = []byte{}
	msg.Meta = []byte("\x73\x75\x70\x65\x72\x2d\x73\x65\x63\x72\x65\x74")
	msg.Version = proto.Uint32(1)

	messageHash := msg.Hash(pubsubTopic)
	require.Equal(t, "0x58e2fc032a82c4adeb967a8b87086d0d6fb304912f120d4404e6236add8f1f56", messageHash.String())

	msg.Timestamp = proto.Int64(123456789123456789)
	messageHash = msg.Hash(pubsubTopic)
	require.Equal(t, "0x978ccc9a665029f9829d42d84e3a49ad3a4791cce53fb5a8b581ef43ad6b4d2f", messageHash.String())
}

func TestIntToBytes(t *testing.T) {
	require.Equal(t, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x27, 0x10}, toBytes(10000))
	require.Equal(t, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x98, 0x96, 0x80}, toBytes(10000000))
}
