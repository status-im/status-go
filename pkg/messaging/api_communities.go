package messaging

import (
	"context"
	"crypto/ecdsa"

	encryption "github.com/status-im/status-go/pkg/messaging/layers/encryption"
	"github.com/status-im/status-go/pkg/messaging/layers/transport"
	types "github.com/status-im/status-go/pkg/messaging/types"
)

func (a *API) SubscribeToPubsubTopic(topic string) error {
	return a.core.stack.Transport.SubscribeToPubsubTopic(topic)
}

func (a *API) GetKeysForGroup(groupID []byte) ([]*encryption.HashRatchetKeyCompatibility, error) {
	return a.core.stack.Encryption.GetKeysForGroup(groupID)
}

func (a *API) GetCurrentKeyForGroup(groupID []byte) (*encryption.HashRatchetKeyCompatibility, error) {
	return a.core.stack.Encryption.GetCurrentKeyForGroup(groupID)
}

func (a *API) SaveHashRatchetMessage(groupID []byte, keyID []byte, m *types.ReceivedMessage) error {
	return a.core.controller.SaveHashRatchetMessage(groupID, keyID, m)
}

func (a *API) GenerateHashRatchetKey(groupID []byte) error {
	return a.core.generateHashRatchetKey(groupID)
}

func (a *API) GetHashRatchetMessagesCountForGroup(groupID []byte) (int, error) {
	return a.core.controller.GetHashRatchetMessagesCountForGroup(groupID)
}

func (a *API) GetAllHRKeysMarshaledV1(groupID []byte) ([]byte, error) {
	return a.core.stack.Encryption.GetAllHRKeysMarshaledV1(groupID)
}

func (a *API) GetAllHRKeysMarshaledV2(groupID []byte) ([]byte, error) {
	return a.core.stack.Encryption.GetAllHRKeysMarshaledV2(groupID)
}

func (a *API) EncryptWithHashRatchet(groupID []byte, payload []byte) ([]byte, []byte, uint32, error) {
	return a.core.encryptWithHashRatchet(groupID, payload)
}

func (a *API) DecryptWithHashRatchet(keyID []byte, seqNo uint32, payload []byte) ([]byte, error) {
	data, err := a.core.stack.Encryption.DecryptWithHashRatchet(keyID, seqNo, payload)
	if err == encryption.ErrNoRatchetKey {
		return nil, types.ErrNoRatchetKey
	}
	return data, err
}

func (a *API) BuildHashRatchetMessage(groupID []byte, payload []byte) ([]byte, error) {
	return a.core.buildHashRatchetMessage(groupID, payload)
}

func (a *API) EncryptCommunityGrants(privateKey *ecdsa.PrivateKey, recipientGrants map[*ecdsa.PublicKey][]byte) (map[uint32][]byte, error) {
	return a.core.stack.Encryption.EncryptCommunityGrants(privateKey, recipientGrants)
}

func (a *API) DecryptCommunityGrant(myIdentityKey *ecdsa.PrivateKey, senderKey *ecdsa.PublicKey, grants map[uint32][]byte) ([]byte, error) {
	return a.core.stack.Encryption.DecryptCommunityGrant(myIdentityKey, senderKey, grants)
}

func (a *API) HandleHashRatchetKeysPayload(ctx context.Context, groupID, encodedKeys []byte, myIdentityKey *ecdsa.PrivateKey, theirIdentityKey *ecdsa.PublicKey) error {
	_, err := a.core.stack.Encryption.HandleHashRatchetKeysPayload(ctx, groupID, encodedKeys, myIdentityKey, theirIdentityKey)
	return err
}

func (a *API) HandleHashRatchetHeadersPayload(ctx context.Context, encodedHeaders [][]byte) error {
	return a.core.stack.Encryption.HandleHashRatchetHeadersPayload(ctx, encodedHeaders)
}

func ToContentTopic(s string) []byte {
	return transport.ToTopic(s)
}

func (a *API) DecryptMessage(myIdentityKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, data []byte) ([]byte, error) {
	data, err := a.core.decryptMessage(myIdentityKey, theirPublicKey, data)
	if err == encryption.ErrHashRatchetGroupIDNotFound {
		return nil, types.ErrHashRatchetGroupIDNotFound
	}
	return data, err
}
