package communities

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
)

func (s *ManagerSuite) TestGetCommunityMembers() {
	community, err := s.manager.CreateCommunity(&requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}, false)
	s.Require().NoError(err)

	ownerKey := crypto.PubkeyToHex(&s.manager.identity.PublicKey)
	s.manager.SetMemberProfileProvider(memberProfilesStub{ownerKey: {DisplayName: "owner"}})

	snapshot, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(community.Clock(), snapshot.Clock)
	s.Require().Len(snapshot.Members, 1)
	s.Require().Equal(ownerKey, snapshot.Members[0].PubKey)
	s.Require().Equal("owner", snapshot.Members[0].DisplayName)

	member, err := crypto.GenerateKey()
	s.Require().NoError(err)
	community, err = s.manager.GetByID(community.ID())
	s.Require().NoError(err)
	_, err = community.AddMember(&member.PublicKey, nil, 0)
	s.Require().NoError(err)
	s.Require().NoError(s.manager.SaveCommunity(community))

	updated, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)
	s.Require().Len(updated.Members, 2)
	s.Require().Greater(updated.Clock, snapshot.Clock)
}

func (s *ManagerSuite) TestMemberDeltaClockFollowsSnapshot() {
	community, err := s.manager.CreateCommunity(&requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}, false)
	s.Require().NoError(err)

	snapshot, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)

	member, err := crypto.GenerateKey()
	s.Require().NoError(err)
	community, err = s.manager.GetByID(community.ID())
	s.Require().NoError(err)
	changes, err := community.AddMember(&member.PublicKey, nil, 0)
	s.Require().NoError(err)
	s.Require().NoError(s.manager.SaveCommunity(community))

	updated, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)

	var delta struct {
		Clock uint64 `json:"clock"`
	}
	bytes, err := json.Marshal(changes)
	s.Require().NoError(err)
	s.Require().NoError(json.Unmarshal(bytes, &delta))
	s.Require().Greater(delta.Clock, snapshot.Clock)
	s.Require().Equal(updated.Clock, delta.Clock)
}

func (s *ManagerSuite) TestCommunityChangesUseMemberProfiles() {
	memberKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	memberHex := crypto.PubkeyToHex(&memberKey.PublicKey)
	s.manager.SetMemberProfileProvider(memberProfilesStub{memberHex: {DisplayName: "member"}})

	community, err := s.manager.CreateCommunity(&requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}, false)
	s.Require().NoError(err)

	stored, err := s.manager.GetByID(community.ID())
	s.Require().NoError(err)

	for _, c := range []*Community{community, stored, stored.CreateDeepCopy()} {
		changes, err := c.AddMember(&memberKey.PublicKey, nil, 0)
		s.Require().NoError(err)

		var delta struct {
			RosterAdded map[string]RosterEntry `json:"rosterAdded"`
		}
		bytes, err := json.Marshal(changes)
		s.Require().NoError(err)
		s.Require().NoError(json.Unmarshal(bytes, &delta))
		s.Require().Equal("member", delta.RosterAdded[memberHex].DisplayName)
	}
}

func (s *ManagerSuite) TestGetCommunityMembersAfterDelete() {
	community, err := s.manager.CreateCommunity(&requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}, false)
	s.Require().NoError(err)

	_, err = s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)

	s.Require().NoError(s.manager.DeleteCommunity(community.ID()))

	_, err = s.manager.GetCommunityMembers(community.ID())
	s.Require().ErrorIs(err, ErrOrgNotFound)
}

func (s *ManagerSuite) createRosterCommunity() *Community {
	community, err := s.manager.CreateCommunity(&requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}, false)
	s.Require().NoError(err)
	return community
}

func (s *ManagerSuite) addRosterMember(communityID []byte) *Community {
	member, err := crypto.GenerateKey()
	s.Require().NoError(err)
	community, err := s.manager.GetByID(communityID)
	s.Require().NoError(err)
	_, err = community.AddMember(&member.PublicKey, nil, 0)
	s.Require().NoError(err)
	return community
}

// A read between the cache invalidation and the write must not cache the
// row being replaced.
func (s *ManagerSuite) TestSaveCommunityReadDuringWrite() {
	community := s.addRosterMember(s.createRosterCommunity().ID())

	s.manager.testHooks.beforeCommunityWrite = func() {
		_, err := s.manager.GetCommunityMembers(community.ID())
		s.Require().NoError(err)
	}
	s.Require().NoError(s.manager.SaveCommunity(community))
	s.manager.testHooks.beforeCommunityWrite = nil

	snapshot, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)
	s.Require().Len(snapshot.Members, 2)
}

// A read that loaded the row before a write must not cache it after the
// write invalidated the cache.
func (s *ManagerSuite) TestSaveCommunityDuringRead() {
	community := s.addRosterMember(s.createRosterCommunity().ID())

	s.manager.testHooks.afterReadonlyLoad = func() {
		s.manager.testHooks.afterReadonlyLoad = nil
		s.Require().NoError(s.manager.SaveCommunity(community))
	}
	_, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)

	snapshot, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)
	s.Require().Len(snapshot.Members, 2)
}

func (s *ManagerSuite) TestDeleteCommunityReadDuringWrite() {
	community := s.createRosterCommunity()

	s.manager.testHooks.beforeCommunityWrite = func() {
		_, err := s.manager.GetCommunityMembers(community.ID())
		s.Require().NoError(err)
	}
	s.Require().NoError(s.manager.DeleteCommunity(community.ID()))
	s.manager.testHooks.beforeCommunityWrite = nil

	_, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().ErrorIs(err, ErrOrgNotFound)
}

func (s *ManagerSuite) rosterDeltaClock(changes *CommunityChanges) uint64 {
	var delta struct {
		Clock uint64 `json:"clock"`
	}
	bytes, err := json.Marshal(changes)
	s.Require().NoError(err)
	s.Require().NoError(json.Unmarshal(bytes, &delta))
	return delta.Clock
}

// A description older than the community events applied locally clears
// them; its member deltas must still be ordered after those events.
func (s *ManagerSuite) TestRosterClockStaysMonotonicWhenEventsAreCleared() {
	community := s.addRosterMember(s.createRosterCommunity().ID())
	s.Require().NoError(s.manager.SaveCommunity(community))
	prevClock := community.Clock()
	eventClock := prevClock + 100
	community.config.EventsData = &EventsData{Events: []CommunityEvent{{CommunityEventClock: eventClock}}}

	snapshot := community.MembersSnapshot(nil)
	s.Require().Equal(eventClock, snapshot.Clock)

	removal := proto.Clone(community.Description()).(*protobuf.CommunityDescription)
	var removedKey string
	for pk := range removal.Members {
		if pk != crypto.PubkeyToHex(&s.manager.identity.PublicKey) {
			removedKey = pk
		}
	}
	delete(removal.Members, removedKey)
	removal.Clock = prevClock + 50

	response, err := s.manager.handleCommunityDescriptionMessageCommon(community, removal, nil, nil)
	s.Require().NoError(err)
	s.Require().Contains(response.Changes.MembersRemoved, removedKey)
	removalClock := s.rosterDeltaClock(response.Changes)
	s.Require().Greater(removalClock, snapshot.Clock)

	after, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(removalClock, after.Clock)

	// A later description still below the cleared events orders after the
	// removal.
	community, err = s.manager.GetByID(community.ID())
	s.Require().NoError(err)
	readd := proto.Clone(removal).(*protobuf.CommunityDescription)
	readd.Members[removedKey] = &protobuf.CommunityMember{}
	readd.Clock = prevClock + 60

	response, err = s.manager.handleCommunityDescriptionMessageCommon(community, readd, nil, nil)
	s.Require().NoError(err)
	s.Require().Contains(response.Changes.MembersAdded, removedKey)
	s.Require().Greater(s.rosterDeltaClock(response.Changes), removalClock)
}

// The cached row joins the community events; writing them must not leave a
// snapshot read before the write in the cache.
func (s *ManagerSuite) TestSaveCommunityEventsInvalidatesSnapshot() {
	community := s.addRosterMember(s.createRosterCommunity().ID())
	s.Require().NoError(s.manager.SaveCommunity(community))

	_, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)

	eventClock := community.Clock() + 100
	community.config.EventsData = &EventsData{
		EventsBaseCommunityDescription: []byte("base"),
		Events:                         []CommunityEvent{{CommunityEventClock: eventClock}},
	}
	s.Require().NoError(s.manager.saveCommunityEvents(community))

	snapshot, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(eventClock, snapshot.Clock)

	s.Require().NoError(s.manager.deleteCommunityEvents(community.ID()))

	snapshot, err = s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(community.Clock(), snapshot.Clock)
}

// setUpEventsReceiver turns the community into one this node does not
// control, with two admins, a plain member and one channel per admin.
func (s *ManagerSuite) setUpEventsReceiver() (*Community, []*ecdsa.PrivateKey, string, []string) {
	community := s.createRosterCommunity()
	var admins []*ecdsa.PrivateKey
	var channels []string
	for i := 0; i < 2; i++ {
		admin, err := crypto.GenerateKey()
		s.Require().NoError(err)
		_, err = community.AddMember(&admin.PublicKey, []protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_ADMIN}, 0)
		s.Require().NoError(err)
		admins = append(admins, admin)

		channelID := fmt.Sprintf("channel-%d", i)
		_, err = community.CreateChat(channelID, &protobuf.CommunityChat{
			Identity:    &protobuf.ChatIdentity{DisplayName: channelID},
			Permissions: &protobuf.CommunityPermissions{Access: protobuf.CommunityPermissions_AUTO_ACCEPT},
			Members:     map[string]*protobuf.CommunityMember{},
		})
		s.Require().NoError(err)
		channels = append(channels, channelID)
	}
	member, err := crypto.GenerateKey()
	s.Require().NoError(err)
	_, err = community.AddMember(&member.PublicKey, nil, 0)
	s.Require().NoError(err)

	raw, err := community.ToProtocolMessageBytes()
	s.Require().NoError(err)
	community.config.CommunityDescriptionProtocolMessage = raw
	community.config.PrivateKey = nil
	s.Require().NoError(s.manager.SaveCommunity(community))

	community, err = s.manager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().False(community.IsControlNode())
	return community, admins, crypto.PubkeyToHex(&member.PublicKey), channels
}

// channelMemberEventsMessage is an admin's channel edit adding member to the
// channel.
func (s *ManagerSuite) channelMemberEventsMessage(community *Community, admin *ecdsa.PrivateKey, channelID string, member string, clock uint64) *protobuf.CommunityEventsMessage {
	channel := proto.Clone(community.Description().Chats[channelID]).(*protobuf.CommunityChat)
	channel.Members = map[string]*protobuf.CommunityMember{member: {}}
	event := CommunityEvent{
		CommunityEventClock: clock,
		Type:                protobuf.CommunityEvent_COMMUNITY_CHANNEL_EDIT,
		ChannelData:         &protobuf.ChannelData{ChannelId: channelID, Channel: channel},
	}
	payload, err := proto.Marshal(event.ToProtobuf())
	s.Require().NoError(err)
	event.Payload = payload
	s.Require().NoError(event.Sign(admin))

	message := &CommunityEventsMessage{
		CommunityID:                    community.ID(),
		EventsBaseCommunityDescription: community.config.CommunityDescriptionProtocolMessage,
		Events:                         []CommunityEvent{event},
	}
	return message.ToProtobuf()
}

// An admin event with a lower clock than one already applied, as from an
// admin with clock skew, still orders its roster delta after the previous one.
func (s *ManagerSuite) TestRosterClockStaysMonotonicForSkewedEvents() {
	community, admins, member, channels := s.setUpEventsReceiver()
	firstClock := community.Clock() + 100

	response, err := s.manager.HandleCommunityEventsMessage(&admins[0].PublicKey, s.channelMemberEventsMessage(community, admins[0], channels[0], member, firstClock))
	s.Require().NoError(err)
	s.Require().True(response.Changes.HasRosterChanges())
	firstDelta := s.rosterDeltaClock(response.Changes)
	s.Require().Equal(firstClock, firstDelta)

	response, err = s.manager.HandleCommunityEventsMessage(&admins[1].PublicKey, s.channelMemberEventsMessage(community, admins[1], channels[1], member, firstClock-5))
	s.Require().NoError(err)
	s.Require().True(response.Changes.HasRosterChanges())
	secondDelta := s.rosterDeltaClock(response.Changes)
	s.Require().Greater(secondDelta, firstDelta)

	snapshot, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(secondDelta, snapshot.Clock)
}

func (s *ManagerSuite) TestRosterClockFloorDroppedOnDeleteAndLeave() {
	deleted := s.createRosterCommunity()
	s.manager.rosterClockFloors.Store(deleted.IDString(), deleted.Clock()+100)
	s.Require().NoError(s.manager.DeleteCommunity(deleted.ID()))
	s.Require().Zero(s.manager.rosterClockFloor(deleted.IDString()))

	left := s.createRosterCommunity()
	s.manager.rosterClockFloors.Store(left.IDString(), left.Clock()+100)
	_, err := s.manager.LeaveCommunity(left.ID())
	s.Require().NoError(err)
	s.Require().Zero(s.manager.rosterClockFloor(left.IDString()))
}

// A reader decoding the row not yet saved must not pair it with the raised
// clock floor, or its snapshot would match the next delta's clock.
func (s *ManagerSuite) TestRosterClockFloorPublishedAfterDescriptionSave() {
	community := s.addRosterMember(s.createRosterCommunity().ID())
	s.Require().NoError(s.manager.SaveCommunity(community))
	prevClock := community.Clock()
	community.config.EventsData = &EventsData{Events: []CommunityEvent{{CommunityEventClock: prevClock + 100}}}

	removal := proto.Clone(community.Description()).(*protobuf.CommunityDescription)
	for pk := range removal.Members {
		if pk != crypto.PubkeyToHex(&s.manager.identity.PublicKey) {
			delete(removal.Members, pk)
		}
	}
	removal.Clock = prevClock + 50

	var during *CommunityMembers
	s.manager.testHooks.beforeCommunityWrite = func() {
		s.manager.testHooks.beforeCommunityWrite = nil
		var err error
		during, err = s.manager.GetCommunityMembers(community.ID())
		s.Require().NoError(err)
	}
	response, err := s.manager.handleCommunityDescriptionMessageCommon(community, removal, nil, nil)
	s.Require().NoError(err)
	s.Require().NotNil(during)
	s.Require().Less(during.Clock, s.rosterDeltaClock(response.Changes))
}

func (s *ManagerSuite) TestRosterClockFloorPublishedAfterEventsSave() {
	community, admins, member, channels := s.setUpEventsReceiver()
	firstClock := community.Clock() + 100

	_, err := s.manager.HandleCommunityEventsMessage(&admins[0].PublicKey, s.channelMemberEventsMessage(community, admins[0], channels[0], member, firstClock))
	s.Require().NoError(err)

	var during *CommunityMembers
	s.manager.testHooks.beforeCommunityWrite = func() {
		s.manager.testHooks.beforeCommunityWrite = nil
		var err error
		during, err = s.manager.GetCommunityMembers(community.ID())
		s.Require().NoError(err)
	}
	response, err := s.manager.HandleCommunityEventsMessage(&admins[1].PublicKey, s.channelMemberEventsMessage(community, admins[1], channels[1], member, firstClock-5))
	s.Require().NoError(err)
	s.Require().NotNil(during)
	s.Require().Less(during.Clock, s.rosterDeltaClock(response.Changes))
}
