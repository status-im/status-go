package communities

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestCommunityBloomFilter(t *testing.T) {
	suite.Run(t, new(CommunityBloomFilterSuite))
}

type CommunityBloomFilterSuite struct {
	suite.Suite
}

func (s *CommunityBloomFilterSuite) TestBasic() {
	ownerIdentity, err := crypto.GenerateKey()
	s.Require().NoError(err)

	memberIdentity, err := crypto.GenerateKey()
	s.Require().NoError(err)

	nonMemberIdentity, err := crypto.GenerateKey()
	s.Require().NoError(err)

	communityID := "cid"
	encryptedChannelID := "enc"
	nonEncryptedChannelID := "non-enc"

	description := &protobuf.CommunityDescription{
		ID:    communityID,
		Clock: 1,
		Chats: map[string]*protobuf.CommunityChat{
			encryptedChannelID: {
				Members: map[string]*protobuf.CommunityMember{
					common.PubkeyToHex(&memberIdentity.PublicKey): {},
				},
			},
			nonEncryptedChannelID: {
				Members: map[string]*protobuf.CommunityMember{
					common.PubkeyToHex(&memberIdentity.PublicKey): {},
				},
			},
		},
		TokenPermissions: map[string]*protobuf.CommunityTokenPermission{
			"permissionID": {
				Id:            "permissionID",
				Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
				TokenCriteria: []*protobuf.TokenCriteria{{}},
				ChatIds:       []string{ChatID(communityID, encryptedChannelID)},
			},
		},
	}

	err = generateBloomFiltersForChannels(description, ownerIdentity)
	s.Require().NoError(err)
	s.Require().NotNil(description.Chats[encryptedChannelID].MembersList)
	s.Require().Nil(description.Chats[nonEncryptedChannelID].MembersList)

	filter := description.Chats[encryptedChannelID].MembersList
	s.Require().True(verifyMembershipWithBloomFilter(filter, memberIdentity, &ownerIdentity.PublicKey, encryptedChannelID, description.Clock))
	s.Require().False(verifyMembershipWithBloomFilter(filter, nonMemberIdentity, &ownerIdentity.PublicKey, encryptedChannelID, description.Clock))
}
