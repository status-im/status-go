package relay

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/hash"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"go.uber.org/zap"
	proto "google.golang.org/protobuf/proto"
)

// Application level message hash
func MsgHash(pubSubTopic string, msg *pb.WakuMessage) []byte {
	timestampBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestampBytes, uint64(msg.Timestamp))

	var ephemeralByte byte
	if msg.Ephemeral {
		ephemeralByte = 1
	}

	return hash.SHA256(
		[]byte(pubSubTopic),
		msg.Payload,
		[]byte(msg.ContentTopic),
		timestampBytes,
		[]byte{ephemeralByte},
	)
}

const MessageWindowDuration = time.Minute * 5

func withinTimeWindow(t timesource.Timesource, msg *pb.WakuMessage) bool {
	if msg.Timestamp == 0 {
		return false
	}

	now := t.Now()
	msgTime := time.Unix(0, msg.Timestamp)

	return now.Sub(msgTime).Abs() <= MessageWindowDuration
}

type validatorFn = func(ctx context.Context, peerID peer.ID, message *pubsub.Message) bool

func validatorFnBuilder(t timesource.Timesource, address common.Address) (validatorFn, error) {
	topic := protocol.NewNamedShardingPubsubTopic(address.String() + "/proto").String()
	return func(ctx context.Context, peerID peer.ID, message *pubsub.Message) bool {
		msg := new(pb.WakuMessage)
		err := proto.Unmarshal(message.Data, msg)
		if err != nil {
			return false
		}

		if !withinTimeWindow(t, msg) {
			return false
		}

		msgHash := MsgHash(topic, msg)
		signature := msg.Meta

		pubKey, err := crypto.SigToPub(msgHash, signature)
		if err != nil {
			return false
		}

		msgAddress := crypto.PubkeyToAddress(*pubKey)

		return bytes.Equal(msgAddress.Bytes(), address.Bytes())
	}, nil
}

func (w *WakuRelay) AddSignedTopicValidator(topic string, address common.Address) error {
	w.log.Info("adding validator to signed topic", zap.String("topic", topic), zap.String("address", address.String()))

	fn, err := validatorFnBuilder(w.timesource, address)
	if err != nil {
		return err
	}

	err = w.pubsub.RegisterTopicValidator(topic, fn)
	if err != nil {
		return err
	}

	if !w.IsSubscribed(topic) {
		w.log.Warn("relay is not subscribed to signed topic", zap.String("topic", topic))
	}

	return nil
}

func SignMessage(privKey *ecdsa.PrivateKey, msg *pb.WakuMessage) error {
	topic := PrivKeyToTopic(privKey)
	msgHash := MsgHash(topic, msg)
	sign, err := secp256k1.Sign(msgHash, crypto.FromECDSA(privKey))
	if err != nil {
		return err
	}

	msg.Meta = sign
	return nil
}

func PrivKeyToTopic(privKey *ecdsa.PrivateKey) string {
	address := crypto.PubkeyToAddress(privKey.PublicKey)
	return protocol.NewNamedShardingPubsubTopic(address.String() + "/proto").String()
}
