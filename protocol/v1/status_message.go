package protocol

import (
	"crypto/ecdsa"
	"encoding/json"
	"log"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/datasync"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/encryption/sharedsecret"
	"github.com/status-im/status-go/protocol/protobuf"
)

type StatusMessageT int

// StatusMessage is any Status Protocol message.
type StatusMessage struct {
	// TransportMessage is the parsed message received from the transport layer, i.e the input
	TransportMessage *types.Message `json:"transportMessage"`
	// Type is the type of application message contained
	Type protobuf.ApplicationMetadataMessage_Type `json:"-"`
	// ParsedMessage is the parsed message by the application layer, i.e the output
	ParsedMessage *reflect.Value `json:"-"`

	// TransportPayload is the payload as received from the transport layer
	TransportPayload []byte `json:"-"`
	// DecryptedPayload is the payload after having been processed by the encryption layer
	DecryptedPayload []byte `json:"decryptedPayload"`
	// UnwrappedPayload is the payload after having been unwrapped from the applicaition metadata layer
	UnwrappedPayload []byte `json:"unwrappedPayload"`

	// ID is the canonical ID of the message
	ID types.HexBytes `json:"id"`
	// Hash is the transport layer hash
	Hash []byte `json:"-"`

	// Dst is the targeted public key
	Dst *ecdsa.PublicKey

	// TransportLayerSigPubKey contains the public key provided by the transport layer
	TransportLayerSigPubKey *ecdsa.PublicKey `json:"-"`
	// ApplicationMetadataLayerPubKey contains the public key provided by the application metadata layer
	ApplicationMetadataLayerSigPubKey *ecdsa.PublicKey `json:"-"`

	// Installations is the new installations returned by the encryption layer
	Installations []*multidevice.Installation
	// SharedSecret is the shared secret returned by the encryption layer
	SharedSecrets []*sharedsecret.Secret
}

// Temporary JSON marshaling for those messages that are not yet processed
// by the go code
func (m *StatusMessage) MarshalJSON() ([]byte, error) {
	item := struct {
		ID        types.HexBytes `json:"id"`
		Payload   string         `json:"payload"`
		From      types.HexBytes `json:"from"`
		Timestamp uint32         `json:"timestamp"`
	}{
		ID:        m.ID,
		Payload:   string(m.UnwrappedPayload),
		Timestamp: m.TransportMessage.Timestamp,
		From:      m.TransportMessage.Sig,
	}
	return json.Marshal(item)
}

// SigPubKey returns the most important signature, from the application layer to transport
func (m *StatusMessage) SigPubKey() *ecdsa.PublicKey {
	if m.ApplicationMetadataLayerSigPubKey != nil {
		return m.ApplicationMetadataLayerSigPubKey
	}

	return m.TransportLayerSigPubKey
}

func (m *StatusMessage) Clone() (*StatusMessage, error) {
	copy := &StatusMessage{}

	err := copier.Copy(&copy, m)
	return copy, err
}

func (m *StatusMessage) HandleTransport(shhMessage *types.Message) error {
	publicKey, err := crypto.UnmarshalPubkey(shhMessage.Sig)
	if err != nil {
		return errors.Wrap(err, "failed to get signature")
	}

	m.TransportMessage = shhMessage
	m.Hash = shhMessage.Hash
	m.TransportLayerSigPubKey = publicKey
	m.TransportPayload = shhMessage.Payload

	if shhMessage.Dst != nil {
		publicKey, err := crypto.UnmarshalPubkey(shhMessage.Dst)
		if err != nil {
			return err
		}
		m.Dst = publicKey
	}

	return nil
}

func (m *StatusMessage) HandleEncryption(myKey *ecdsa.PrivateKey, senderKey *ecdsa.PublicKey, enc *encryption.Protocol, skipNegotiation bool) error {
	// As we handle non-encrypted messages, we make sure that DecryptPayload
	// is set regardless of whether this step is successful
	m.DecryptedPayload = m.TransportPayload
	// Nothing to do
	if skipNegotiation {
		return nil
	}

	var protocolMessage encryption.ProtocolMessage
	err := proto.Unmarshal(m.TransportPayload, &protocolMessage)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ProtocolMessage")
	}

	response, err := enc.HandleMessage(
		myKey,
		senderKey,
		&protocolMessage,
		m.Hash,
	)

	if err != nil {
		return errors.Wrap(err, "failed to handle Encryption message")
	}

	m.DecryptedPayload = response.DecryptedMessage
	m.Installations = response.Installations
	m.SharedSecrets = response.SharedSecrets
	return nil
}

// HandleDatasync processes StatusMessage through data sync layer.
// This is optional and DataSync might be nil. In such a case,
// only one payload will be returned equal to DecryptedPayload.
func (m *StatusMessage) HandleDatasync(datasync *datasync.DataSync) ([]*StatusMessage, error) {
	var statusMessages []*StatusMessage

	payloads := datasync.Handle(
		m.SigPubKey(),
		m.DecryptedPayload,
	)

	for _, payload := range payloads {
		message, err := m.Clone()
		if err != nil {
			return nil, err
		}
		message.DecryptedPayload = payload
		statusMessages = append(statusMessages, message)
	}
	return statusMessages, nil
}

func (m *StatusMessage) HandleApplicationMetadata() error {
	message, err := protobuf.Unmarshal(m.DecryptedPayload)
	if err != nil {
		return err
	}

	recoveredKey, err := message.RecoverKey()
	if err != nil {
		return err
	}
	m.ApplicationMetadataLayerSigPubKey = recoveredKey
	// Calculate ID using the wrapped record
	m.ID = MessageID(recoveredKey, m.DecryptedPayload)
	m.UnwrappedPayload = message.Payload
	m.Type = message.Type
	return nil

}

func (m *StatusMessage) HandleApplication() error {
	switch m.Type {
	case protobuf.ApplicationMetadataMessage_CHAT_MESSAGE:
		return m.unmarshalProtobufData(new(protobuf.ChatMessage))

	case protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE:
		return m.unmarshalProtobufData(new(protobuf.MembershipUpdateMessage))

	case protobuf.ApplicationMetadataMessage_ACCEPT_REQUEST_ADDRESS_FOR_TRANSACTION:
		return m.unmarshalProtobufData(new(protobuf.AcceptRequestAddressForTransaction))

	case protobuf.ApplicationMetadataMessage_SEND_TRANSACTION:
		return m.unmarshalProtobufData(new(protobuf.SendTransaction))

	case protobuf.ApplicationMetadataMessage_REQUEST_TRANSACTION:
		return m.unmarshalProtobufData(new(protobuf.RequestTransaction))

	case protobuf.ApplicationMetadataMessage_DECLINE_REQUEST_ADDRESS_FOR_TRANSACTION:
		return m.unmarshalProtobufData(new(protobuf.DeclineRequestAddressForTransaction))

	case protobuf.ApplicationMetadataMessage_DECLINE_REQUEST_TRANSACTION:
		return m.unmarshalProtobufData(new(protobuf.DeclineRequestTransaction))

	case protobuf.ApplicationMetadataMessage_REQUEST_ADDRESS_FOR_TRANSACTION:
		return m.unmarshalProtobufData(new(protobuf.RequestAddressForTransaction))

	case protobuf.ApplicationMetadataMessage_CONTACT_UPDATE:
		return m.unmarshalProtobufData(new(protobuf.ContactUpdate))

	case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION:
		return m.unmarshalProtobufData(new(protobuf.SyncInstallation))

	case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_CONTACT:
		log.Printf("Sync installation contact")
		return m.unmarshalProtobufData(new(protobuf.SyncInstallationContact))

	case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_PUBLIC_CHAT:
		return m.unmarshalProtobufData(new(protobuf.SyncInstallationPublicChat))

	case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_ACCOUNT:
		return m.unmarshalProtobufData(new(protobuf.SyncInstallationAccount))

	case protobuf.ApplicationMetadataMessage_PAIR_INSTALLATION:
		return m.unmarshalProtobufData(new(protobuf.PairInstallation))
	case protobuf.ApplicationMetadataMessage_CONTACT_CODE_ADVERTISEMENT:
		return m.unmarshalProtobufData(new(protobuf.ContactCodeAdvertisement))
	case protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_REQUEST:
		return m.unmarshalProtobufData(new(protobuf.PushNotificationRequest))
	case protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_REGISTRATION_RESPONSE:
		return m.unmarshalProtobufData(new(protobuf.PushNotificationRegistrationResponse))
	case protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_QUERY:
		return m.unmarshalProtobufData(new(protobuf.PushNotificationQuery))
	case protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_QUERY_RESPONSE:
		return m.unmarshalProtobufData(new(protobuf.PushNotificationQueryResponse))
	case protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_RESPONSE:
		return m.unmarshalProtobufData(new(protobuf.PushNotificationResponse))
	case protobuf.ApplicationMetadataMessage_EMOJI_REACTION:
		return m.unmarshalProtobufData(new(protobuf.EmojiReaction))
	case protobuf.ApplicationMetadataMessage_GROUP_CHAT_INVITATION:
		return m.unmarshalProtobufData(new(protobuf.GroupChatInvitation))
	case protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION:
		return m.unmarshalProtobufData(new(protobuf.CommunityDescription))
	case protobuf.ApplicationMetadataMessage_COMMUNITY_INVITATION:
		return m.unmarshalProtobufData(new(protobuf.CommunityInvitation))
	case protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_REGISTRATION:
		// This message is a bit different as it's encrypted, so we pass it straight through
		v := reflect.ValueOf(m.UnwrappedPayload)
		m.ParsedMessage = &v
		return nil
	case protobuf.ApplicationMetadataMessage_CHAT_IDENTITY:
		return m.unmarshalProtobufData(new(protobuf.ChatIdentity))
	}
	return nil
}

func (m *StatusMessage) unmarshalProtobufData(pb proto.Message) error {
	var ptr proto.Message
	rv := reflect.ValueOf(pb)
	if rv.Kind() == reflect.Ptr {
		ptr = pb
	} else {
		ptr = rv.Addr().Interface().(proto.Message)
	}

	err := proto.Unmarshal(m.UnwrappedPayload, ptr)
	if err != nil {
		m.ParsedMessage = nil
		log.Printf("[message::DecodeMessage] could not decode %T: %#x, err: %v", pb, m.Hash, err.Error())
	} else {
		rv = reflect.ValueOf(ptr)
		elem := rv.Elem()
		m.ParsedMessage = &elem
		return nil
	}

	return nil
}
