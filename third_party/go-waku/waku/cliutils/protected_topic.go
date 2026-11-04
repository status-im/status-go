package cliutils

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type ProtectedTopic struct {
	Topic     string
	PublicKey *ecdsa.PublicKey
}

func (p ProtectedTopic) String() string {
	pubKBytes := crypto.FromECDSAPub(p.PublicKey)
	return fmt.Sprintf("%s:%s", p.Topic, hex.EncodeToString(pubKBytes))
}

type ProtectedTopicSlice struct {
	Values *[]ProtectedTopic
}

func (k *ProtectedTopicSlice) Set(value string) error {
	protectedTopicParts := strings.Split(value, ":")
	if len(protectedTopicParts) != 2 {
		return errors.New("expected topic_name:hex_encoded_public_key")
	}

	pubk, err := crypto.UnmarshalPubkey(common.FromHex(protectedTopicParts[1]))
	if err != nil {
		return err
	}
	*k.Values = append(*k.Values, ProtectedTopic{
		Topic:     protectedTopicParts[0],
		PublicKey: pubk,
	})
	return nil
}

func (k *ProtectedTopicSlice) String() string {
	if k.Values == nil {
		return ""
	}
	var output []string
	for _, v := range *k.Values {
		output = append(output, v.String())
	}

	return strings.Join(output, ", ")
}
