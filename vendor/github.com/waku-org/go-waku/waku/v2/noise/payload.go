package noise

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"errors"

	n "github.com/waku-org/noise"
)

const MaxUint8 = 1<<8 - 1

// This follows https://rfc.vac.dev/spec/35/#public-keys-serialization
// pk contains the X coordinate of the public key, if unencrypted (this implies flag = 0)
// or the encryption of the X coordinate concatenated with the authorization tag, if encrypted (this implies flag = 1)
// Note: besides encryption, flag can be used to distinguish among multiple supported Elliptic Curves
type NoisePublicKey struct {
	Flag   byte
	PubKey []byte
}

func byteToNoisePublicKey(input []byte) *NoisePublicKey {
	flag := byte(0)
	if len(input) > n.DH25519.DHLen() {
		flag = 1
	}

	return &NoisePublicKey{
		Flag:   flag,
		PubKey: input,
	}
}

// EcdsaPubKeyToNoisePublicKey converts a Elliptic Curve public key
// to an unencrypted Noise public key
func Ed25519PubKeyToNoisePublicKey(pk ed25519.PublicKey) *NoisePublicKey {
	return &NoisePublicKey{
		Flag:   0,
		PubKey: pk,
	}
}

// Equals checks equality between two Noise public keys
func (pk *NoisePublicKey) Equals(pk2 *NoisePublicKey) bool {
	return pk.Flag == pk2.Flag && bytes.Equal(pk.PubKey, pk2.PubKey)
}

type SerializedNoisePublicKey []byte

// Serialize converts a Noise public key to a stream of bytes as in
// https://rfc.vac.dev/spec/35/#public-keys-serialization
func (pk *NoisePublicKey) Serialize() SerializedNoisePublicKey {
	// Public key is serialized as (flag || pk)
	// Note that pk contains the X coordinate of the public key if unencrypted
	// or the encryption concatenated with the authorization tag if encrypted
	serializedPK := make([]byte, len(pk.PubKey)+1)
	serializedPK[0] = pk.Flag
	copy(serializedPK[1:], pk.PubKey)

	return serializedPK
}

// Unserialize converts a serialized Noise public key to a NoisePublicKey object as in
// https://rfc.vac.dev/spec/35/#public-keys-serialization
func (s SerializedNoisePublicKey) Unserialize() (*NoisePublicKey, error) {
	if len(s) <= 1 {
		return nil, errors.New("invalid serialized public key length")
	}

	pubk := &NoisePublicKey{}
	pubk.Flag = s[0]
	if !(pubk.Flag == 0 || pubk.Flag == 1) {
		return nil, errors.New("invalid flag in serialized public key")
	}

	pubk.PubKey = s[1:]

	return pubk, nil
}

// Encrypt encrypts a Noise public key using a Cipher State
func (pk *NoisePublicKey) Encrypt(state *n.CipherState) error {
	if pk.Flag == 0 {
		// Authorization tag is appended to output
		encPk, err := state.Encrypt(nil, nil, pk.PubKey)
		if err != nil {
			return err
		}
		pk.Flag = 1
		pk.PubKey = encPk
	}

	return nil
}

// Decrypts decrypts a Noise public key using a Cipher State
func (pk *NoisePublicKey) Decrypt(state *n.CipherState) error {
	if pk.Flag == 1 {
		decPk, err := state.Decrypt(nil, nil, pk.PubKey) // encrypted pk should contain the auth tag
		if err != nil {
			return err
		}
		pk.Flag = 0
		pk.PubKey = decPk
	}

	return nil
}

// PayloadV2 defines an object for Waku payloads with version 2 as in
// https://rfc.vac.dev/spec/35/#public-keys-serialization
// It contains a protocol ID field, the handshake message (for Noise handshakes) and
// a transport message (for Noise handshakes and ChaChaPoly encryptions)
type PayloadV2 struct {
	ProtocolId       byte
	HandshakeMessage []*NoisePublicKey
	TransportMessage []byte
}

// Checks equality between two PayloadsV2 objects
func (p *PayloadV2) Equals(p2 *PayloadV2) bool {
	if p.ProtocolId != p2.ProtocolId || !bytes.Equal(p.TransportMessage, p2.TransportMessage) {
		return false
	}

	for _, p1 := range p.HandshakeMessage {
		for _, p2 := range p2.HandshakeMessage {
			if !p1.Equals(p2) {
				return false
			}
		}
	}

	return true
}

// Serializes a PayloadV2 object to a byte sequences according to https://rfc.vac.dev/spec/35/
// The output serialized payload concatenates the input PayloadV2 object fields as
// payload = ( protocolId || serializedHandshakeMessageLen || serializedHandshakeMessage || transportMessageLen || transportMessage)
// The output can be then passed to the payload field of a WakuMessage https://rfc.vac.dev/spec/14/
func (p *PayloadV2) Serialize() ([]byte, error) {
	// We collect public keys contained in the handshake message

	// According to https://rfc.vac.dev/spec/35/, the maximum size for the handshake message is 256 bytes, that is
	// the handshake message length can be represented with 1 byte only. (its length can be stored in 1 byte)
	// However, to ease public keys length addition operation, we declare it as int and later cast to uit8
	serializedHandshakeMessageLen := 0
	// This variables will store the concatenation of the serializations of all public keys in the handshake message
	serializedHandshakeMessage := make([]byte, 0, 256)
	serializedHandshakeMessageBuffer := bytes.NewBuffer(serializedHandshakeMessage)

	for _, pk := range p.HandshakeMessage {
		serializedPK := pk.Serialize()
		serializedHandshakeMessageLen += len(serializedPK)
		if _, err := serializedHandshakeMessageBuffer.Write(serializedPK); err != nil {
			return nil, err
		}
		if serializedHandshakeMessageLen > MaxUint8 {
			return nil, errors.New("too many public keys in handshake message")
		}
	}

	// The output payload as in https://rfc.vac.dev/spec/35/. We concatenate all the PayloadV2 fields as
	// payload = ( protocolId || serializedHandshakeMessageLen || serializedHandshakeMessage || transportMessageLen || transportMessage)

	// We declare it as a byte sequence of length accordingly to the PayloadV2 information read
	payload := make([]byte, 0, 1+ // 1 byte for protocol ID
		1+ // 1 byte for length of serializedHandshakeMessage field
		serializedHandshakeMessageLen+ // serializedHandshakeMessageLen bytes for serializedHandshakeMessage
		8+ // 8 bytes for transportMessageLen
		len(p.TransportMessage), // transportMessageLen bytes for transportMessage
	)

	payloadBuf := bytes.NewBuffer(payload)

	//  The protocol ID (1 byte) and handshake message length (1 byte) can be directly casted to byte to allow direct copy to the payload byte sequence
	if err := payloadBuf.WriteByte(p.ProtocolId); err != nil {
		return nil, err
	}

	if err := payloadBuf.WriteByte(byte(serializedHandshakeMessageLen)); err != nil {
		return nil, err
	}

	if _, err := payloadBuf.Write(serializedHandshakeMessageBuffer.Bytes()); err != nil {
		return nil, err
	}

	TransportMessageLen := uint64(len(p.TransportMessage))
	if err := binary.Write(payloadBuf, binary.LittleEndian, TransportMessageLen); err != nil {
		return nil, err
	}

	if _, err := payloadBuf.Write(p.TransportMessage); err != nil {
		return nil, err
	}

	return payloadBuf.Bytes(), nil
}

func isProtocolIDSupported(protocolID WakuNoiseProtocolID) bool {
	return protocolID == Noise_K1K1_25519_ChaChaPoly_SHA256 || protocolID == Noise_XK1_25519_ChaChaPoly_SHA256 ||
		protocolID == Noise_XX_25519_ChaChaPoly_SHA256 || protocolID == Noise_XXpsk0_25519_ChaChaPoly_SHA256 ||
		protocolID == ChaChaPoly || protocolID == None
}

const ChaChaPolyTagSize = byte(16)

// Deserializes a byte sequence to a PayloadV2 object according to https://rfc.vac.dev/spec/35/.
// The input serialized payload concatenates the output PayloadV2 object fields as
// payload = ( protocolId || serializedHandshakeMessageLen || serializedHandshakeMessage || transportMessageLen || transportMessage)
func DeserializePayloadV2(payload []byte) (*PayloadV2, error) {
	payloadBuf := bytes.NewBuffer(payload)

	result := &PayloadV2{}

	// We start reading the Protocol ID
	// TODO: when the list of supported protocol ID is defined, check if read protocol ID is supported
	if err := binary.Read(payloadBuf, binary.BigEndian, &result.ProtocolId); err != nil {
		return nil, err
	}

	if !isProtocolIDSupported(result.ProtocolId) {
		return nil, errors.New("unsupported protocol")
	}

	// We read the Handshake Message length (1 byte)
	var handshakeMessageLen byte
	if err := binary.Read(payloadBuf, binary.BigEndian, &handshakeMessageLen); err != nil {
		return nil, err
	}
	if handshakeMessageLen > MaxUint8 {
		return nil, errors.New("too many public keys in handshake message")
	}

	written := byte(0)
	var handshakeMessages []*NoisePublicKey
	for written < handshakeMessageLen {
		// We obtain the current Noise Public key encryption flag
		flag, err := payloadBuf.ReadByte()
		if err != nil {
			return nil, err
		}

		if flag == 0 {
			// If the key is unencrypted, we only read the X coordinate of the EC public key and we deserialize into a Noise Public Key
			pkLen := ed25519.PublicKeySize
			var pkBytes SerializedNoisePublicKey = make([]byte, pkLen)
			if err := binary.Read(payloadBuf, binary.BigEndian, &pkBytes); err != nil {
				return nil, err
			}

			serializedPK := SerializedNoisePublicKey(make([]byte, ed25519.PublicKeySize+1))
			serializedPK[0] = flag
			copy(serializedPK[1:], pkBytes)

			pk, err := serializedPK.Unserialize()
			if err != nil {
				return nil, err
			}

			handshakeMessages = append(handshakeMessages, pk)
			written += uint8(len(serializedPK))

		} else if flag == 1 {
			// If the key is encrypted, we only read the encrypted X coordinate and the authorization tag, and we deserialize into a Noise Public Key
			pkLen := ed25519.PublicKeySize + ChaChaPolyTagSize
			// TODO: duplicated code: ==============

			var pkBytes SerializedNoisePublicKey = make([]byte, pkLen)
			if err := binary.Read(payloadBuf, binary.BigEndian, &pkBytes); err != nil {
				return nil, err
			}

			serializedPK := SerializedNoisePublicKey(make([]byte, ed25519.PublicKeySize+1))
			serializedPK[0] = flag
			copy(serializedPK[1:], pkBytes)

			pk, err := serializedPK.Unserialize()
			if err != nil {
				return nil, err
			}

			handshakeMessages = append(handshakeMessages, pk)
			written += uint8(len(serializedPK))
			// TODO: duplicated
		} else {
			return nil, errors.New("invalid flag for Noise public key")
		}
	}

	result.HandshakeMessage = handshakeMessages

	var TransportMessageLen uint64
	if err := binary.Read(payloadBuf, binary.LittleEndian, &TransportMessageLen); err != nil {
		return nil, err
	}

	result.TransportMessage = make([]byte, TransportMessageLen)
	if err := binary.Read(payloadBuf, binary.BigEndian, &result.TransportMessage); err != nil {
		return nil, err
	}

	return result, nil
}
