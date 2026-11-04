package library

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"google.golang.org/protobuf/proto"
)

func wakuMessage(messageJSON string) (*pb.WakuMessage, error) {
	var msg *pb.WakuMessage
	err := json.Unmarshal([]byte(messageJSON), &msg)
	return msg, err
}

func EncodeSymmetric(messageJSON string, symmetricKey string, optionalSigningKey string) (string, error) {
	msg, err := wakuMessage(messageJSON)
	if err != nil {
		return "", err
	}

	msgPayload := payload.Payload{
		Data: msg.Payload,
		Key: &payload.KeyInfo{
			Kind: payload.Symmetric,
		},
	}

	keyBytes, err := utils.DecodeHexString(symmetricKey)
	if err != nil {
		return "", err
	}

	msgPayload.Key.SymKey = keyBytes

	if optionalSigningKey != "" {
		signingKeyBytes, err := utils.DecodeHexString(optionalSigningKey)
		if err != nil {
			return "", err
		}

		msgPayload.Key.PrivKey, err = crypto.ToECDSA(signingKeyBytes)
		if err != nil {
			return "", err
		}
	}

	msg.Version = proto.Uint32(payload.V1Encryption)
	msg.Payload, err = msgPayload.Encode(1)
	if err != nil {
		return "", err
	}

	encodedMsg, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	return string(encodedMsg), err
}

func EncodeAsymmetric(messageJSON string, publicKey string, optionalSigningKey string) (string, error) {
	msg, err := wakuMessage(messageJSON)
	if err != nil {
		return "", err
	}

	msgPayload := payload.Payload{
		Data: msg.Payload,
		Key: &payload.KeyInfo{
			Kind: payload.Asymmetric,
		},
	}

	keyBytes, err := utils.DecodeHexString(publicKey)
	if err != nil {
		return "", err
	}

	msgPayload.Key.PubKey, err = unmarshalPubkey(keyBytes)
	if err != nil {
		return "", err
	}

	if optionalSigningKey != "" {
		signingKeyBytes, err := utils.DecodeHexString(optionalSigningKey)
		if err != nil {
			return "", err
		}

		msgPayload.Key.PrivKey, err = crypto.ToECDSA(signingKeyBytes)
		if err != nil {
			return "", err
		}
	}

	msg.Version = proto.Uint32(payload.V1Encryption)
	msg.Payload, err = msgPayload.Encode(payload.V1Encryption)
	if err != nil {
		return "", err
	}

	encodedMsg, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	return string(encodedMsg), err
}

func extractPubKeyAndSignature(payload *payload.DecodedPayload) (pubkey string, signature string) {
	pkBytes := crypto.FromECDSAPub(payload.PubKey)
	if len(pkBytes) != 0 {
		pubkey = hexutil.Encode(pkBytes)
	}

	if len(payload.Signature) != 0 {
		signature = hexutil.Encode(payload.Signature)
	}

	return
}

type v0Response struct {
	Data    []byte `json:"data"`
	Padding []byte `json:"padding"`
}

type v1Response struct {
	PubKey    string `json:"pubkey,omitempty"`
	Signature string `json:"signature,omitempty"`
	Data      []byte `json:"data"`
	Padding   []byte `json:"padding"`
}

// DecodeSymmetric decodes a waku message using a 32 bytes symmetric key. The key must be a hex encoded string with "0x" prefix
func DecodeSymmetric(messageJSON string, symmetricKey string) (string, error) {
	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(messageJSON), &msg)
	if err != nil {
		return "", err
	}

	if msg.GetVersion() == payload.Unencrypted {
		return marshalJSON(v0Response{
			Data: msg.Payload,
		})
	} else if msg.GetVersion() != payload.V1Encryption {
		return "", errors.New("unsupported wakumessage version")
	}

	keyInfo := &payload.KeyInfo{
		Kind: payload.Symmetric,
	}

	keyInfo.SymKey, err = utils.DecodeHexString(symmetricKey)
	if err != nil {
		return "", err
	}

	payload, err := payload.DecodePayload(&msg, keyInfo)
	if err != nil {
		return "", err
	}

	pubkey, signature := extractPubKeyAndSignature(payload)

	response := v1Response{
		PubKey:    pubkey,
		Signature: signature,
		Data:      payload.Data,
		Padding:   payload.Padding,
	}

	return marshalJSON(response)
}

// DecodeAsymmetric decodes a waku message using a secp256k1 private key. The key must be a hex encoded string with "0x" prefix
func DecodeAsymmetric(messageJSON string, privateKey string) (string, error) {
	var msg pb.WakuMessage
	err := json.Unmarshal([]byte(messageJSON), &msg)
	if err != nil {
		return "", err
	}

	if msg.GetVersion() == payload.Unencrypted {
		return marshalJSON(v0Response{
			Data: msg.Payload,
		})
	} else if msg.GetVersion() != payload.V1Encryption {
		return "", errors.New("unsupported wakumessage version")
	}

	keyInfo := &payload.KeyInfo{
		Kind: payload.Asymmetric,
	}

	keyBytes, err := utils.DecodeHexString(privateKey)
	if err != nil {
		return "", err
	}

	keyInfo.PrivKey, err = crypto.ToECDSA(keyBytes)
	if err != nil {
		return "", err
	}

	payload, err := payload.DecodePayload(&msg, keyInfo)
	if err != nil {
		return "", err
	}

	pubkey, signature := extractPubKeyAndSignature(payload)

	response := v1Response{
		PubKey:    pubkey,
		Signature: signature,
		Data:      payload.Data,
		Padding:   payload.Padding,
	}

	return marshalJSON(response)
}

func unmarshalPubkey(pub []byte) (ecdsa.PublicKey, error) {
	x, y := secp256k1.S256().Unmarshal(pub)
	if x == nil {
		return ecdsa.PublicKey{}, errors.New("invalid public key")
	}
	return ecdsa.PublicKey{Curve: secp256k1.S256(), X: x, Y: y}, nil
}
