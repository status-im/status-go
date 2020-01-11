package protocol

import (
	"crypto/ecdsa"
	"encoding/json"
	"log"

	"github.com/golang/protobuf/proto"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/datasync"
	"github.com/status-im/status-go/protocol/encryption"
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
	ParsedMessage interface{} `json:"-"`

	// TransportPayload is the payload as received from the transport layer
	TransportPayload []byte `json:"-"`
	// DecryptedPayload is the payload after having been processed by the encryption layer
	DecryptedPayload []byte `json:"decryptedPayload"`

	// ID is the canonical ID of the message
	ID types.HexBytes `json:"id"`
	// Hash is the transport layer hash
	Hash []byte `json:"-"`

	// TransportLayerSigPubKey contains the public key provided by the transport layer
	TransportLayerSigPubKey *ecdsa.PublicKey `json:"-"`
	// ApplicationMetadataLayerPubKey contains the public key provided by the application metadata layer
	ApplicationMetadataLayerSigPubKey *ecdsa.PublicKey `json:"-"`
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
		Payload:   string(m.DecryptedPayload),
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

	return nil
}

func (m *StatusMessage) HandleEncryption(myKey *ecdsa.PrivateKey, senderKey *ecdsa.PublicKey, enc *encryption.Protocol) error {
	// As we handle non-encrypted messages, we make sure that DecryptPayload
	// is set regardless of whether this step is successful
	m.DecryptedPayload = m.TransportPayload

	var protocolMessage encryption.ProtocolMessage
	err := proto.Unmarshal(m.TransportPayload, &protocolMessage)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ProtocolMessage")
	}

	payload, err := enc.HandleMessage(
		myKey,
		senderKey,
		&protocolMessage,
		m.Hash,
	)

	if err != nil {
		return errors.Wrap(err, "failed to handle Encryption message")
	}

	m.DecryptedPayload = payload
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
	m.DecryptedPayload = message.Payload
	m.Type = message.Type
	return nil

}

func (m *StatusMessage) HandleApplication() error {
	switch m.Type {
	case protobuf.ApplicationMetadataMessage_CHAT_MESSAGE:
		var message protobuf.ChatMessage

		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode ChatMessage: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}
	case protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE:
		var message protobuf.MembershipUpdateMessage
		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode MembershipUpdateMessage: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}
	case protobuf.ApplicationMetadataMessage_ACCEPT_REQUEST_ADDRESS_FOR_TRANSACTION:
		var message protobuf.AcceptRequestAddressForTransaction
		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode AcceptRequestAddressForTransaction: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}
	case protobuf.ApplicationMetadataMessage_SEND_TRANSACTION:
		var message protobuf.SendTransaction
		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode SendTransaction: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}

	case protobuf.ApplicationMetadataMessage_REQUEST_TRANSACTION:
		var message protobuf.RequestTransaction
		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode RequestTransaction: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}

	case protobuf.ApplicationMetadataMessage_DECLINE_REQUEST_ADDRESS_FOR_TRANSACTION:
		var message protobuf.DeclineRequestAddressForTransaction
		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode DeclineRequestAddressForTransaction: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}
	case protobuf.ApplicationMetadataMessage_DECLINE_REQUEST_TRANSACTION:
		var message protobuf.DeclineRequestTransaction
		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode DeclineRequestTransaction: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}

	case protobuf.ApplicationMetadataMessage_REQUEST_ADDRESS_FOR_TRANSACTION:
		var message protobuf.RequestAddressForTransaction
		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode RequestAddressForTransaction: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}

	case protobuf.ApplicationMetadataMessage_CONTACT_UPDATE:
		var message protobuf.ContactUpdate
		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode ContactUpdate: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}
	case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION:
		var message protobuf.SyncInstallation
		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode SyncInstallation: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}
	case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_CONTACT:
		var message protobuf.SyncInstallationContact
		log.Printf("Sync installation contact")
		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode SyncInstallationContact: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}
	case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_PUBLIC_CHAT:
		var message protobuf.SyncInstallationPublicChat
		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode SyncInstallationPublicChat: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}
	case protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_ACCOUNT:
		var message protobuf.SyncInstallationAccount
		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode SyncInstallationAccount: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}
	case protobuf.ApplicationMetadataMessage_PAIR_INSTALLATION:
		var message protobuf.PairInstallation
		err := proto.Unmarshal(m.DecryptedPayload, &message)
		if err != nil {
			m.ParsedMessage = nil
			log.Printf("[message::DecodeMessage] could not decode PairInstallation: %#x, err: %v", m.Hash, err.Error())
		} else {
			m.ParsedMessage = message

			return nil
		}
	}
	return nil
}
