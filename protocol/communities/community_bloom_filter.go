package communities

import (
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"math/bits"

	"github.com/bits-and-blooms/bloom/v3"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/protobuf"
)

func generateBloomFiltersForChannels(description *protobuf.CommunityDescription, privateKey *ecdsa.PrivateKey) error {
	for channelID, channel := range description.Chats {
		if !channelEncrypted(ChatID(description.ID, channelID), description.TokenPermissions) {
			continue
		}

		filter, err := generateBloomFilter(channel.Members, privateKey, channelID, description.Clock)
		if err != nil {
			return err
		}

		marshaledFilter, err := filter.MarshalBinary()
		if err != nil {
			return err
		}

		channel.MembersList = &protobuf.CommunityBloomFilter{
			Data: marshaledFilter,
			M:    uint64(filter.Cap()),
			K:    uint64(filter.K()),
		}
	}

	return nil
}

func nextPowerOfTwo(x int) uint {
	return 1 << bits.Len(uint(x))
}

func max(x, y uint) uint {
	if x > y {
		return x
	}
	return y
}

func generateBloomFilter(members map[string]*protobuf.CommunityMember, privateKey *ecdsa.PrivateKey, channelID string, clock uint64) (*bloom.BloomFilter, error) {
	membersCount := len(members)
	if membersCount == 0 {
		return nil, errors.New("invalid members count")
	}

	const falsePositiveRate = 0.001
	numberOfItems := max(128, nextPowerOfTwo(membersCount)) // This makes it difficult to guess the exact number of members, even with knowledge of filter size and parameters.
	filter := bloom.NewWithEstimates(numberOfItems, falsePositiveRate)

	for pk := range members {
		publicKey, err := common.HexToPubkey(pk)
		if err != nil {
			return nil, err
		}

		value, err := bloomFilterValue(privateKey, publicKey, channelID, clock)
		if err != nil {
			return nil, err
		}

		filter.Add(value)
	}

	return filter, nil
}

func verifyMembershipWithBloomFilter(membersList *protobuf.CommunityBloomFilter, privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey, channelID string, clock uint64) (bool, error) {
	filter := bloom.New(uint(membersList.M), uint(membersList.K))
	err := filter.UnmarshalBinary(membersList.Data)
	if err != nil {
		return false, err
	}

	value, err := bloomFilterValue(privateKey, publicKey, channelID, clock)
	if err != nil {
		return false, err
	}

	return filter.Test(value), nil
}

func bloomFilterValue(privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey, channelID string, clock uint64) ([]byte, error) {
	sharedSecret, err := encryption.GenerateSharedKey(privateKey, publicKey)
	if err != nil {
		return nil, err
	}

	clockBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(clockBytes, clock)

	return crypto.Keccak256(sharedSecret, []byte(channelID), clockBytes), nil
}
