package main

import (
	whisper "github.com/status-im/whisper/whisperv6"
	"golang.org/x/crypto/sha3"
)

func nameToTopic(name string) (whisper.TopicType, error) {
	hash := sha3.NewLegacyKeccak256()
	if _, err := hash.Write([]byte(name)); err != nil {
		return whisper.TopicType{}, err
	}

	return whisper.BytesToTopic(hash.Sum(nil)), nil
}
