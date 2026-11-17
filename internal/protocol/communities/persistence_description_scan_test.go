package communities

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

func communityMembers(count int) map[string]*protobuf.CommunityMember {
	members := make(map[string]*protobuf.CommunityMember, count)
	for i := 0; i < count; i++ {
		members[fmt.Sprintf("0x04%0128x", i)] = &protobuf.CommunityMember{
			Roles: []protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_NONE},
		}
	}
	return members
}

func (s *PersistenceSuite) saveCommunityWithDescription(description *protobuf.CommunityDescription) []byte {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	community := Community{
		config: &Config{
			PrivateKey:           key,
			ControlNode:          &key.PublicKey,
			ControlDevice:        true,
			ID:                   &key.PublicKey,
			CommunityDescription: description,
		},
	}
	s.Require().NoError(s.db.SaveCommunity(&community))
	return community.ID()
}

func (s *PersistenceSuite) TestCommunityClockOfUnknownCommunity() {
	clock, found, err := s.db.CommunityClock([]byte{0x02, 0x01})
	s.Require().NoError(err)
	s.Require().False(found)
	s.Require().Zero(clock)
}

func (s *PersistenceSuite) TestCommunityClockOfStoredCommunity() {
	id := s.saveCommunityWithDescription(&protobuf.CommunityDescription{
		Clock:   1763300000123,
		Members: communityMembers(2000),
	})

	clock, found, err := s.db.CommunityClock(id)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(uint64(1763300000123), clock)
}

func (s *PersistenceSuite) TestCommunityClockZero() {
	id := s.saveCommunityWithDescription(&protobuf.CommunityDescription{
		Members: communityMembers(2000),
	})

	clock, found, err := s.db.CommunityClock(id)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Zero(clock)
}

// A description is valid protobuf in any field order: the clock may follow the members.
func (s *PersistenceSuite) TestCommunityClockAfterLargeFields() {
	members, err := proto.Marshal(&protobuf.CommunityDescription{Members: communityMembers(2000)})
	s.Require().NoError(err)
	payload := protowire.AppendVarint(protowire.AppendTag(members, 1, protowire.VarintType), 42)

	wrapped := protowire.AppendBytes(protowire.AppendTag(nil, 1, protowire.BytesType), make([]byte, 65))
	wrapped = protowire.AppendBytes(protowire.AppendTag(wrapped, 2, protowire.BytesType), payload)
	wrapped = protowire.AppendVarint(protowire.AppendTag(wrapped, 3, protowire.VarintType),
		uint64(protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION))

	description, err := decodeWrappedCommunityDescription(wrapped)
	s.Require().NoError(err)
	s.Require().Equal(uint64(42), description.Clock)
	s.Require().Len(description.Members, 2000)

	id := []byte{0x02, 0x02}
	s.Require().NoError(s.db.saveRawCommunityRowWithoutSyncedAt(RawCommunityRow{ID: id, Description: wrapped}))

	clock, found, err := s.db.CommunityClock(id)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(uint64(42), clock)
}

func (s *PersistenceSuite) wrappedDescription(description *protobuf.CommunityDescription) []byte {
	id := s.saveCommunityWithDescription(description)
	row, err := s.db.getRawCommunityRow(id)
	s.Require().NoError(err)
	return row.Description
}

func (s *PersistenceSuite) TestCommunityClockOfMalformedDescription() {
	valid := s.wrappedDescription(&protobuf.CommunityDescription{
		Clock:   7,
		Members: communityMembers(2000),
	})
	clockLast, err := proto.Marshal(&protobuf.CommunityDescription{Members: communityMembers(2000)})
	s.Require().NoError(err)
	clockLast = protowire.AppendVarint(protowire.AppendTag(clockLast, 1, protowire.VarintType), 42)
	wrappedClockLast := protowire.AppendBytes(protowire.AppendTag(nil, 2, protowire.BytesType), clockLast)

	malformed := map[string][]byte{
		"not protobuf":                    {0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		"cut before the clock":            valid[:70],
		"cut inside the members":          wrappedClockLast[:len(wrappedClockLast)/2],
		"cut inside the trailing clock":   wrappedClockLast[:len(wrappedClockLast)-1],
		"payload longer than description": protowire.AppendVarint(protowire.AppendTag(nil, 2, protowire.BytesType), 100000),
	}

	i := byte(0)
	for name, description := range malformed {
		i++
		id := []byte{0x02, 0x10, i}
		s.Require().NoError(s.db.saveRawCommunityRowWithoutSyncedAt(RawCommunityRow{ID: id, Description: description}), name)

		_, found, err := s.db.CommunityClock(id)
		s.Require().ErrorIs(err, errMalformedDescription, name)
		s.Require().False(found, name)
	}
}
