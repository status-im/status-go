package relay

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/binary"
	"encoding/hex"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/hash"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"go.uber.org/zap"
	proto "google.golang.org/protobuf/proto"
)

func msgHash(pubSubTopic string, msg *pb.WakuMessage) []byte {
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

const messageWindowDuration = time.Minute * 5

func withinTimeWindow(t timesource.Timesource, msg *pb.WakuMessage) bool {
	if msg.Timestamp == 0 {
		return false
	}

	now := t.Now()
	msgTime := time.Unix(0, msg.Timestamp)

	return now.Sub(msgTime).Abs() <= messageWindowDuration
}

type validatorFn = func(ctx context.Context, peerID peer.ID, message *pubsub.Message) bool

func validatorFnBuilder(t timesource.Timesource, topic string, publicKey *ecdsa.PublicKey) (validatorFn, error) {
	publicKeyBytes := crypto.FromECDSAPub(publicKey)
	return func(ctx context.Context, peerID peer.ID, message *pubsub.Message) bool {
		msg := new(pb.WakuMessage)
		err := proto.Unmarshal(message.Data, msg)
		if err != nil {
			return false
		}

		if !withinTimeWindow(t, msg) {
			return false
		}

		msgHash := msgHash(topic, msg)
		signature := msg.Meta

		return secp256k1.VerifySignature(publicKeyBytes, msgHash, signature)
	}, nil
}

// AddSignedTopicValidator registers a gossipsub validator for a topic which will check that messages Meta field contains a valid ECDSA signature for the specified pubsub topic. This is used as a DoS prevention mechanism
func (w *WakuRelay) AddSignedTopicValidator(topic string, publicKey *ecdsa.PublicKey) error {
	w.log.Info("adding validator to signed topic", zap.String("topic", topic), zap.String("publicKey", hex.EncodeToString(elliptic.Marshal(publicKey.Curve, publicKey.X, publicKey.Y))))

	fn, err := validatorFnBuilder(w.timesource, topic, publicKey)
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

// SignMessage adds an ECDSA signature to a WakuMessage as an opt-in mechanism for DoS prevention
func SignMessage(privKey *ecdsa.PrivateKey, msg *pb.WakuMessage, pubsubTopic string) error {
	msgHash := msgHash(pubsubTopic, msg)
	sign, err := secp256k1.Sign(msgHash, crypto.FromECDSA(privKey))
	if err != nil {
		return err
	}

	msg.Meta = sign[0:64] // Remove V
	return nil
}
