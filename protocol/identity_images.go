package protocol

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"errors"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func EncryptIdentityImagesWithContactPubKeys(iis map[string]*protobuf.IdentityImage, m *Messenger) error {
	// Make AES key
	AESKey := make([]byte, 32)
	_, err := crand.Read(AESKey)
	if err != nil {
		return err
	}

	for _, ii := range iis {
		// Encrypt image payload with the AES key
		encryptedPayload, err := common.Encrypt(ii.Payload, AESKey, crand.Reader)
		if err != nil {
			return err
		}

		// Overwrite the unencrypted payload with the newly encrypted payload
		ii.Payload = encryptedPayload
		ii.Encrypted = true
		for _, c := range m.allContacts {
			if !c.IsAdded() {
				continue
			}

			pubK, err := c.PublicKey()
			if err != nil {
				return err
			}
			// Generate a Diffie-Helman (DH) between the sender private key and the recipient's public key
			sharedKey, err := common.MakeECDHSharedKey(m.identity, pubK)
			if err != nil {
				return err
			}

			// Encrypt the main AES key with AES encryption using the DH key
			eAESKey, err := common.Encrypt(AESKey, sharedKey, crand.Reader)
			if err != nil {
				return err
			}

			// Append the the encrypted main AES key to the IdentityImage's EncryptionKeys slice.
			ii.EncryptionKeys = append(ii.EncryptionKeys, eAESKey)
		}
	}

	return nil
}

func DecryptIdentityImagesWithIdentityPrivateKey(iis map[string]*protobuf.IdentityImage, recipientIdentity *ecdsa.PrivateKey, senderPubKey *ecdsa.PublicKey) error {
image:
	for _, ii := range iis {
		for _, empk := range ii.EncryptionKeys {
			// Generate a Diffie-Helman (DH) between the recipient's private key and the sender's public key
			sharedKey, err := common.MakeECDHSharedKey(recipientIdentity, senderPubKey)
			if err != nil {
				return err
			}

			// Decrypt the main encryption AES key with AES encryption using the DH key
			dAESKey, err := common.Decrypt(empk, sharedKey)
			if err != nil {
				if err.Error() == "cipher: message authentication failed" {
					continue
				}
				return err
			}
			if dAESKey == nil{
				return errors.New("decrypting the payload encryption key resulted in no error and a nil key")
			}

			// Decrypt the payload with the newly decrypted main encryption AES key
			payload, err := common.Decrypt(ii.Payload, dAESKey)
			if err != nil {
				return err
			}
			if payload == nil {
				// TODO should this be a logger warn? A payload could theoretically be validly empty
				return errors.New("decrypting the payload resulted in no error and a nil payload")
			}

			// Overwrite the payload with the decrypted data
			ii.Payload = payload
			ii.Encrypted = false
			continue image
		}
	}

	return nil
}

