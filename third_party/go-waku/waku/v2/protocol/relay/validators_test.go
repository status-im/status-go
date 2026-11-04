package relay

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	proto "google.golang.org/protobuf/proto"
)

type FakeTimesource struct {
	now time.Time
}

func NewFakeTimesource(now time.Time) FakeTimesource {
	return FakeTimesource{now}
}

func (f FakeTimesource) Start(ctx context.Context) error {
	return nil
}

func (f FakeTimesource) Stop() {}

func (f FakeTimesource) Now() time.Time {
	return f.now
}

type Timesource interface {
	Now() time.Time
}

func TestMsgHash(t *testing.T) {
	privKeyB, _ := hex.DecodeString("5526a8990317c9b7b58d07843d270f9cd1d9aaee129294c1c478abf7261dd9e6")
	prvKey, _ := crypto.ToECDSA(privKeyB)

	payload, _ := hex.DecodeString("1A12E077D0E89F9CAC11FBBB6A676C86120B5AD3E248B1F180E98F15EE43D2DFCF62F00C92737B2FF6F59B3ABA02773314B991C41DC19ADB0AD8C17C8E26757B")
	protectedPubSubTopic := "pubsub-topic"
	contentTopic := "content-topic"
	ephemeral := true
	timestamp := time.Unix(0, 1683208172339052800)

	msg := &pb.WakuMessage{
		Payload:      payload,
		ContentTopic: contentTopic,
		Timestamp:    proto.Int64(timestamp.UnixNano()),
		Ephemeral:    proto.Bool(ephemeral),
	}

	err := SignMessage(prvKey, msg, protectedPubSubTopic)
	require.NoError(t, err)

	//	expectedSignature, _ := hex.DecodeString("127FA211B2514F0E974A055392946DC1A14052182A6ABEFB8A6CD7C51DA1BF2E40595D28EF1A9488797C297EED3AAC45430005FB3A7F037BDD9FC4BD99F59E63")
	//	require.True(t, bytes.Equal(expectedSignature, msg.Meta))

	//expectedMessageHash, _ := hex.DecodeString("662F8C20A335F170BD60ABC1F02AD66F0C6A6EE285DA2A53C95259E7937C0AE9")
	//messageHash := MsgHash(pubsubTopic, msg)
	//require.True(t, bytes.Equal(expectedMessageHash, messageHash))

	myValidator := signedTopicBuilder(NewFakeTimesource(timestamp), &prvKey.PublicKey)
	result := myValidator(context.Background(), msg, protectedPubSubTopic)
	require.True(t, result)

	// Exceed 5m window in both directions
	now5m1sInPast := timestamp.Add(-5 * time.Minute).Add(-1 * time.Second)
	myValidator = signedTopicBuilder(NewFakeTimesource(now5m1sInPast), &prvKey.PublicKey)
	require.NoError(t, err)
	result = myValidator(context.Background(), msg, protectedPubSubTopic)
	require.False(t, result)

	now5m1sInFuture := timestamp.Add(5 * time.Minute).Add(1 * time.Second)
	myValidator = signedTopicBuilder(NewFakeTimesource(now5m1sInFuture), &prvKey.PublicKey)
	result = myValidator(context.Background(), msg, protectedPubSubTopic)
	require.False(t, result)
}
