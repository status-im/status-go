package communities

import (
	"crypto/ecdsa"
	"runtime"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

func identityWithImages(images map[string][]byte) *protobuf.ChatIdentity {
	identity := &protobuf.ChatIdentity{
		DisplayName: "status",
		Images:      map[string]*protobuf.IdentityImage{},
	}
	for name, payload := range images {
		identity.Images[name] = &protobuf.IdentityImage{
			Payload:     payload,
			SourceType:  protobuf.IdentityImage_RAW_PAYLOAD,
			ImageFormat: protobuf.ImageFormat_JPEG,
		}
	}
	return identity
}

// storeReceivedCommunity stores a community the way one received from the
// network is stored: signed by its control node, not controlled by us.
func (s *ManagerSuite) storeReceivedCommunity(description *protobuf.CommunityDescription, events ...CommunityEvent) (string, *ecdsa.PrivateKey) {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	controlNode, err := New(Config{
		MemberIdentity:       s.manager.identity,
		PrivateKey:           key,
		ControlNode:          &key.PublicKey,
		ControlDevice:        true,
		ID:                   &key.PublicKey,
		CommunityDescription: description,
	}, &TimeSourceStub{}, nil, nil)
	s.Require().NoError(err)
	wrappedDescription, err := controlNode.ToProtocolMessageBytes()
	s.Require().NoError(err)

	community, err := New(Config{
		MemberIdentity:                      s.manager.identity,
		ControlNode:                         &key.PublicKey,
		ID:                                  &key.PublicKey,
		CommunityDescription:                description,
		CommunityDescriptionProtocolMessage: wrappedDescription,
	}, &TimeSourceStub{}, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(s.manager.persistence.SaveCommunity(community))

	if len(events) > 0 {
		community.config.EventsData = &EventsData{
			EventsBaseCommunityDescription: wrappedDescription,
			Events:                         events,
		}
		s.Require().NoError(s.manager.persistence.SaveCommunityEvents(community))
	}

	return community.IDString(), key
}

func (s *ManagerSuite) TestCommunityImagesOfStoredCommunity() {
	communityID, _ := s.storeReceivedCommunity(&protobuf.CommunityDescription{
		Clock:   7,
		Members: communityMembers(100),
		Identity: identityWithImages(map[string][]byte{
			"thumbnail": []byte("thumbnail-bytes"),
			"banner":    []byte("banner-bytes"),
		}),
	})

	images, err := s.manager.communityImages(communityID)
	s.Require().NoError(err)
	s.Require().Len(images, 2)
	s.Require().Equal([]byte("thumbnail-bytes"), images["thumbnail"].Payload)
	s.Require().Equal([]byte("banner-bytes"), images["banner"].Payload)
	s.Require().Equal(protobuf.ImageFormat_JPEG, images["banner"].ImageFormat)
}

func (s *ManagerSuite) TestCommunityImagesOfCommunityWithoutImages() {
	communityID, _ := s.storeReceivedCommunity(&protobuf.CommunityDescription{Clock: 7})

	images, err := s.manager.communityImages(communityID)
	s.Require().NoError(err)
	s.Require().NotNil(images)
	s.Require().Empty(images)
}

func (s *ManagerSuite) TestCommunityImagesOfUnknownCommunity() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	_, err = s.manager.communityImages(types.EncodeHex(crypto.CompressPubkey(&key.PublicKey)))
	s.Require().ErrorIs(err, ErrOrgNotFound)

	_, err = s.manager.communityImages("not-hex")
	s.Require().Error(err)
}

// An admin's pending edit replaces the identity of the stored description.
func (s *ManagerSuite) TestCommunityImagesFollowPendingEdit() {
	communityID, _ := s.storeReceivedCommunity(&protobuf.CommunityDescription{
		Clock:    7,
		Identity: identityWithImages(map[string][]byte{"thumbnail": []byte("stored-bytes")}),
	}, CommunityEvent{
		CommunityEventClock: 8,
		Type:                protobuf.CommunityEvent_COMMUNITY_EDIT,
		CommunityConfig: &protobuf.CommunityConfig{
			Identity: identityWithImages(map[string][]byte{"thumbnail": []byte("edited-bytes")}),
		},
	})

	images, err := s.manager.communityImages(communityID)
	s.Require().NoError(err)
	s.Require().Equal([]byte("edited-bytes"), images["thumbnail"].Payload)
}

func (s *ManagerSuite) TestCommunityImagesDoNotLoadCommunity() {
	const requests = 20
	image := make([]byte, 64*1024)
	communityID, _ := s.storeReceivedCommunity(&protobuf.CommunityDescription{
		Clock:    7,
		Members:  communityMembers(5000),
		Identity: identityWithImages(map[string][]byte{"thumbnail": image, "banner": image}),
	})

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < requests; i++ {
		images, err := s.manager.communityImages(communityID)
		s.Require().NoError(err)
		s.Require().Len(images, 2)
	}
	runtime.ReadMemStats(&after)

	perRequest := (after.TotalAlloc - before.TotalAlloc) / requests
	s.T().Logf("allocated per image request: %d bytes", perRequest)
	s.Require().Less(perRequest, uint64(2*1024*1024))
}

func (s *ManagerSuite) TestCommunityExists() {
	communityID, _ := s.storeReceivedCommunity(&protobuf.CommunityDescription{Clock: 7})
	id, err := types.DecodeHex(communityID)
	s.Require().NoError(err)

	exists, err := s.manager.CommunityExists(id)
	s.Require().NoError(err)
	s.Require().True(exists)

	exists, err = s.manager.CommunityExists([]byte{0x02, 0x01})
	s.Require().NoError(err)
	s.Require().False(exists)
}

func (s *ManagerSuite) TestCommunityExistsDoesNotReadDescription() {
	const checks = 20
	communityID, _ := s.storeReceivedCommunity(&protobuf.CommunityDescription{
		Clock:   7,
		Members: communityMembers(5000),
	})
	id, err := types.DecodeHex(communityID)
	s.Require().NoError(err)

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < checks; i++ {
		exists, err := s.manager.CommunityExists(id)
		s.Require().NoError(err)
		s.Require().True(exists)
	}
	runtime.ReadMemStats(&after)

	perCheck := (after.TotalAlloc - before.TotalAlloc) / checks
	s.T().Logf("allocated per existence check: %d bytes", perCheck)
	// The stored description is over 0.7 MB.
	s.Require().Less(perCheck, uint64(64*1024))
}
