package adapter

import (
	"context"
	"log"
	"time"

	"github.com/status-im/status-console-client/protocol/subscription"
	"github.com/status-im/status-console-client/protocol/transport"
	"github.com/status-im/status-console-client/protocol/v1"

	whisper "github.com/status-im/whisper/whisperv6"

	"github.com/status-im/status-go/services/shhext/chat"
)

type ProtocolWhisperAdapter struct {
	transport transport.WhisperTransport
	pfs       *chat.ProtocolService
}

// ProtocolWhisperAdapter must implement Protocol interface.
var _ protocol.Protocol = (*ProtocolWhisperAdapter)(nil)

func NewProtocolWhisperAdapter(t transport.WhisperTransport, pfs *chat.ProtocolService) *ProtocolWhisperAdapter {
	return &ProtocolWhisperAdapter{
		transport: t,
		pfs:       pfs,
	}
}

// Subscribe listens to new messages.
func (w *ProtocolWhisperAdapter) Subscribe(
	ctx context.Context,
	messages chan<- *protocol.Message,
	options protocol.SubscribeOptions,
) (*subscription.Subscription, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	filter := newFilter(w.transport.KeysManager())
	if err := updateFilterFromSubscribeOptions(filter, options); err != nil {
		return nil, err
	}

	// Messages income in batches and hence a buffered channel is used.
	in := make(chan *whisper.ReceivedMessage, 1024)
	sub, err := w.transport.Subscribe(ctx, in, filter.Filter)
	if err != nil {
		return nil, err
	}

	go func() {
		for item := range in {
			message, err := w.decodeMessage(item)
			if err != nil {
				log.Printf("failed to decode message %#+x: %v", item.EnvelopeHash.Bytes(), err)
				continue
			}
			messages <- message
		}
	}()

	return sub, nil
}

func (w *ProtocolWhisperAdapter) decodeMessage(message *whisper.ReceivedMessage) (*protocol.Message, error) {
	payload := message.Payload
	publicKey := message.SigToPubKey()
	hash := message.EnvelopeHash.Bytes()

	if w.pfs != nil {
		decryptedPayload, err := w.pfs.HandleMessage(
			w.transport.KeysManager().PrivateKey(),
			publicKey,
			payload,
			hash,
		)
		if err != nil {
			log.Printf("failed to handle message %#+x by PFS: %v", hash, err)
		} else {
			payload = decryptedPayload
		}
	}

	decoded, err := protocol.DecodeMessage(payload)
	if err != nil {
		return nil, err
	}
	decoded.ID = hash
	decoded.SigPubKey = publicKey

	return &decoded, nil
}

// Send sends a message to the network.
// Identity is required as the protocol requires
// all messages to be signed.
func (w *ProtocolWhisperAdapter) Send(ctx context.Context, data []byte, options protocol.SendOptions) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	if w.pfs != nil {
		var (
			encryptedData []byte
			err           error
		)

		privateKey := w.transport.KeysManager().PrivateKey()

		// TODO: rethink this
		if options.Recipient != nil {
			encryptedData, err = w.pfs.BuildDirectMessage(
				privateKey,
				options.Recipient,
				data,
			)
		} else {
			encryptedData, err = w.pfs.BuildPublicMessage(privateKey, data)
		}

		if err != nil {
			return nil, err
		}
		data = encryptedData
	}

	newMessage, err := NewNewMessage(w.transport.KeysManager(), data)
	if err != nil {
		return nil, err
	}
	if err := updateNewMessageFromSendOptions(newMessage, options); err != nil {
		return nil, err
	}

	return w.transport.Send(ctx, newMessage.NewMessage)
}

// Request retrieves historic messages.
func (w *ProtocolWhisperAdapter) Request(ctx context.Context, params protocol.RequestOptions) error {
	transOptions := transport.RequestOptions{
		Password: MailServerPassword,
		Topics:   []whisper.TopicType{},
		From:     params.From,
		To:       params.To,
		Limit:    params.Limit,
	}
	for _, chat := range params.Chats {
		topic, err := ToTopic(chat.ChatName)
		if err != nil {
			return err
		}
		transOptions.Topics = append(transOptions.Topics, topic)
	}
	now := time.Now()
	err := w.transport.Request(ctx, transOptions)
	log.Printf("[ProtocolWhisperAdapter::Request] took %s", time.Since(now))
	return err
}
