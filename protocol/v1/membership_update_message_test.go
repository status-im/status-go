package protocol

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/protobuf"
)

var (
	testMembershipUpdateMessageStruct = MembershipUpdateMessage{
		ChatID: "chat-id",
		Events: []MembershipUpdateEvent{
			{
				Type:       protobuf.MembershipUpdateEvent_CHAT_CREATED,
				Name:       "thathata",
				ChatID:     "chat-id",
				ClockValue: 156897373998501,
			},
			{
				Type:       protobuf.MembershipUpdateEvent_MEMBERS_ADDED,
				Members:    []string{"0x04aebe2bb01a988abe7d978662f21de7760486119876c680e5a559e38e086a2df6dad41c4e4d9079c03db3bced6cb70fca76afc5650e50ea19b81572046a813534"},
				ChatID:     "chat-id",
				ClockValue: 156897373998502,
			},
		},
	}
)

func TestSignMembershipUpdate(t *testing.T) {
	key, err := crypto.HexToECDSA("838fbdd1b670209a258b90af25653a018bc582c44c56e6290a973eebbeb15732")
	require.NoError(t, err)
	event := &testMembershipUpdateMessageStruct.Events[0]
	err = event.Sign(key)
	require.NoError(t, err)

	encodedEvent, err := proto.Marshal(event.ToProtobuf())
	require.NoError(t, err)

	var signatureMaterial []byte
	signatureMaterial = append(signatureMaterial, []byte(testMembershipUpdateMessageStruct.ChatID)...)
	signatureMaterial = crypto.Keccak256(append(signatureMaterial, encodedEvent...))
	expected, err := crypto.Sign(signatureMaterial, key)
	require.NoError(t, err)
	require.Equal(t, encodedEvent, event.RawPayload)
	require.Equal(t, expected, event.Signature)

	// Sign the other event
	err = testMembershipUpdateMessageStruct.Events[1].Sign(key)
	require.NoError(t, err)

	// Encode message
	encodedMessage, err := testMembershipUpdateMessageStruct.ToProtobuf()
	require.NoError(t, err)
	// Verify it
	verifiedMessage, err := MembershipUpdateMessageFromProtobuf(encodedMessage)
	require.NoError(t, err)
	require.Equal(t, verifiedMessage, &testMembershipUpdateMessageStruct)
}

func TestGroupCreator(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	g, err := NewGroupWithCreator("abc", 20, key)
	require.NoError(t, err)
	creator, err := g.creator()
	require.NoError(t, err)
	require.Equal(t, publicKeyToString(&key.PublicKey), creator)
}

func TestGroupProcessEvent(t *testing.T) {
	createGroup := func(admins, members, joined []string, name string) Group {
		return Group{
			name:    name,
			admins:  newStringSetFromSlice(admins),
			joined:  newStringSetFromSlice(joined),
			members: newStringSetFromSlice(members),
		}
	}

	testCases := []struct {
		Name   string
		Group  Group
		Result Group
		From   string
		Event  MembershipUpdateEvent
	}{
		{
			Name:   "chat-created event",
			Group:  createGroup(nil, nil, nil, ""),
			Result: createGroup([]string{"0xabc"}, []string{"0xabc"}, []string{"0xabc"}, "some-name"),
			From:   "0xabc",
			Event:  NewChatCreatedEvent("some-name", 0),
		},
		{
			Name:   "name-changed event",
			Group:  createGroup(nil, nil, nil, ""),
			Result: createGroup(nil, nil, nil, "some-name"),
			From:   "0xabc",
			Event:  NewNameChangedEvent("some-name", 0),
		},
		{
			Name:   "admins-added event",
			Group:  createGroup(nil, nil, nil, ""),
			Result: createGroup([]string{"0xabc", "0x123"}, nil, nil, ""),
			From:   "0xabc",
			Event:  NewAdminsAddedEvent([]string{"0xabc", "0x123"}, 0),
		},
		{
			Name:   "admin-removed event",
			Group:  createGroup([]string{"0xabc", "0xdef"}, nil, nil, ""),
			Result: createGroup([]string{"0xdef"}, nil, nil, ""),
			From:   "0xabc",
			Event:  NewAdminRemovedEvent("0xabc", 0),
		},
		{
			Name:   "members-added event",
			Group:  createGroup(nil, nil, nil, ""),
			Result: createGroup(nil, []string{"0xabc", "0xdef"}, nil, ""),
			From:   "0xabc",
			Event:  NewMembersAddedEvent([]string{"0xabc", "0xdef"}, 0),
		},
		{
			Name:   "member-removed event",
			Group:  createGroup(nil, []string{"0xabc", "0xdef"}, []string{"0xdef", "0xabc"}, ""),
			Result: createGroup(nil, []string{"0xdef"}, []string{"0xdef"}, ""),
			From:   "0xabc",
			Event:  NewMemberRemovedEvent("0xabc", 0),
		},
		{
			Name:   "member-joined event",
			Group:  createGroup(nil, []string{"0xabc", "0xdef"}, []string{"0xabc"}, ""),
			Result: createGroup(nil, []string{"0xabc", "0xdef"}, []string{"0xabc", "0xdef"}, ""),
			From:   "0xdef",
			Event:  NewMemberJoinedEvent(0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			g := tc.Group
			tc.Event.From = tc.From
			g.processEvent(tc.Event)
			require.EqualValues(t, tc.Result, g)
		})
	}
}

func TestGroupValidateEvent(t *testing.T) {
	createGroup := func(admins, members []string) Group {
		return Group{
			admins:  newStringSetFromSlice(admins),
			members: newStringSetFromSlice(members),
		}
	}

	testCases := []struct {
		Name   string
		From   string
		Group  Group
		Event  MembershipUpdateEvent
		Result bool
	}{
		{
			Name:   "chat-created with empty admins and members",
			Group:  createGroup(nil, nil),
			From:   "0xabc",
			Event:  NewChatCreatedEvent("test", 0),
			Result: true,
		},
		{
			Name:   "chat-created with existing admins",
			Group:  createGroup([]string{"0xabc"}, nil),
			From:   "0xabc",
			Event:  NewChatCreatedEvent("test", 0),
			Result: false,
		},
		{
			Name:   "chat-created with existing members",
			Group:  createGroup(nil, []string{"0xabc"}),
			From:   "0xabc",
			Event:  NewChatCreatedEvent("test", 0),
			Result: false,
		},
		{
			Name:   "name-changed allowed because from is admin",
			From:   "0xabc",
			Group:  createGroup([]string{"0xabc"}, nil),
			Event:  NewNameChangedEvent("new-name", 0),
			Result: true,
		},
		{
			Name:   "name-changed not allowed for non-admins",
			From:   "0xabc",
			Group:  createGroup(nil, nil),
			Event:  NewNameChangedEvent("new-name", 0),
			Result: false,
		},
		{
			Name:   "members-added allowed because from is admin",
			From:   "0xabc",
			Group:  createGroup([]string{"0xabc"}, nil),
			Event:  NewMembersAddedEvent([]string{"0x123"}, 0),
			Result: true,
		},
		{
			Name:   "members-added not allowed for non-admins",
			From:   "0xabc",
			Group:  createGroup(nil, nil),
			Event:  NewMembersAddedEvent([]string{"0x123"}, 0),
			Result: false,
		},
		{
			Name:   "member-removed allowed because removing themselves",
			From:   "0xabc",
			Group:  createGroup(nil, nil),
			Event:  NewMemberRemovedEvent("0xabc", 0),
			Result: true,
		},
		{
			Name:   "member-removed allowed because from is admin",
			From:   "0xabc",
			Group:  createGroup([]string{"0xabc"}, nil),
			Event:  NewMemberRemovedEvent("0x123", 0),
			Result: true,
		},
		{
			Name:   "member-removed not allowed for non-admins",
			From:   "0xabc",
			Group:  createGroup(nil, nil),
			Event:  NewMemberRemovedEvent("0x123", 0),
			Result: false,
		},
		{
			Name:   "member-joined must be in members",
			From:   "0xabc",
			Group:  createGroup(nil, []string{"0xabc"}),
			Event:  NewMemberJoinedEvent(0),
			Result: true,
		},
		{
			Name:   "member-joined not valid because not in members",
			From:   "0xabc",
			Group:  createGroup(nil, nil),
			Event:  NewMemberJoinedEvent(0),
			Result: false,
		},
		{
			Name:   "member-joined not valid because from differs from the event",
			From:   "0xdef",
			Group:  createGroup(nil, nil),
			Event:  NewMemberJoinedEvent(0),
			Result: false,
		},
		{
			Name:   "admins-added allowed because originating from other admin",
			From:   "0xabc",
			Group:  createGroup([]string{"0xabc", "0x123"}, []string{"0xdef", "0xghi"}),
			Event:  NewAdminsAddedEvent([]string{"0xdef"}, 0),
			Result: true,
		},
		{
			Name:   "admins-added not allowed because not from admin",
			From:   "0xabc",
			Group:  createGroup([]string{"0x123"}, []string{"0xdef", "0xghi"}),
			Event:  NewAdminsAddedEvent([]string{"0xdef"}, 0),
			Result: false,
		},
		{
			Name:   "admins-added not allowed because not in members",
			From:   "0xabc",
			Group:  createGroup([]string{"0xabc", "0x123"}, []string{"0xghi"}),
			Event:  NewAdminsAddedEvent([]string{"0xdef"}, 0),
			Result: false,
		},
		{
			Name:   "admin-removed allowed because is admin and removes themselves",
			From:   "0xabc",
			Group:  createGroup([]string{"0xabc"}, nil),
			Event:  NewAdminRemovedEvent("0xabc", 0),
			Result: true,
		},
		{
			Name:   "admin-removed not allowed because not themselves",
			From:   "0xabc",
			Group:  createGroup([]string{"0xabc", "0xdef"}, nil),
			Event:  NewAdminRemovedEvent("0xdef", 0),
			Result: false,
		},
		{
			Name:   "admin-removed not allowed because not admin",
			From:   "0xdef",
			Group:  createGroup([]string{"0xabc"}, nil),
			Event:  NewAdminRemovedEvent("0xabc", 0),
			Result: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.Event.From = tc.From
			result := tc.Group.validateEvent(tc.Event)
			assert.Equal(t, tc.Result, result)
		})
	}
}

func TestMembershipUpdateEventEqual(t *testing.T) {
	u1 := MembershipUpdateEvent{
		Type:       protobuf.MembershipUpdateEvent_CHAT_CREATED,
		ClockValue: 1,
		Members:    []string{"0xabc"},
		Name:       "abc",
		Signature:  []byte("signature"),
	}
	require.True(t, u1.Equal(u1))

	// Verify equality breaking.
	u2 := u1
	u2.Signature = []byte("different-signature")
	require.False(t, u1.Equal(u2))
}

func TestAbridgedEvents(t *testing.T) {
	var clock uint64 = 0
	creator, err := crypto.GenerateKey()
	require.NoError(t, err)
	creatorID := publicKeyToString(&creator.PublicKey)

	member1, err := crypto.GenerateKey()
	require.NoError(t, err)
	member1ID := publicKeyToString(&member1.PublicKey)

	member2, err := crypto.GenerateKey()
	require.NoError(t, err)
	member2ID := publicKeyToString(&member2.PublicKey)

	member3, err := crypto.GenerateKey()
	require.NoError(t, err)
	member3ID := publicKeyToString(&member3.PublicKey)

	member4, err := crypto.GenerateKey()
	require.NoError(t, err)
	member4ID := publicKeyToString(&member4.PublicKey)

	g, err := NewGroupWithCreator("name-0", clock, creator)
	require.NoError(t, err)
	clock++

	// Full events is only a single one
	require.Len(t, g.Events(), 1)
	// same as abridged
	require.Len(t, g.AbridgedEvents(&creator.PublicKey), 1)

	// We change name of the chat
	nameChangedEvent1 := NewNameChangedEvent("name-1", clock)
	nameChangedEvent1.From = creatorID
	nameChangedEvent1.ChatID = g.chatID
	err = g.ProcessEvent(nameChangedEvent1)
	require.NoError(t, err)
	clock++

	// We change name of the chat again
	nameChangedEvent2 := NewNameChangedEvent("name-2", clock)
	nameChangedEvent2.From = creatorID
	nameChangedEvent2.ChatID = g.chatID
	err = g.ProcessEvent(nameChangedEvent2)
	require.NoError(t, err)
	clock++

	// Full events is 3 events
	require.Len(t, g.Events(), 3)
	// While abridged should exclude the first name-1 event
	require.Len(t, g.AbridgedEvents(&creator.PublicKey), 2)
	require.Equal(t, g.AbridgedEvents(&creator.PublicKey)[1].Name, "name-2")

	// Add a new member
	newMemberEvent1 := NewMembersAddedEvent([]string{member1ID}, clock)
	newMemberEvent1.From = creatorID
	newMemberEvent1.ChatID = g.chatID
	err = g.ProcessEvent(newMemberEvent1)
	require.NoError(t, err)
	clock++

	// Full events is 4 events
	require.Len(t, g.Events(), 4)
	// While abridged, given we are the creator, we only take 2 events and ignore
	// the member created event
	require.Len(t, g.AbridgedEvents(&creator.PublicKey), 2)
	require.Equal(t, g.AbridgedEvents(&creator.PublicKey)[1].Name, "name-2")

	// While abridged, given we are the new member, we take 3 events
	// that are relevant to us
	require.Len(t, g.AbridgedEvents(&member1.PublicKey), 3)
	require.Equal(t, g.AbridgedEvents(&member1.PublicKey)[1].Name, "name-2")
	require.Equal(t, g.AbridgedEvents(&member1.PublicKey)[2].Members, []string{member1ID})

	// We join the chat
	joinedEvent1 := NewMemberJoinedEvent(clock)
	joinedEvent1.From = member1ID
	joinedEvent1.ChatID = g.chatID
	err = g.ProcessEvent(joinedEvent1)
	require.NoError(t, err)
	clock++

	// Full events is 5 events
	require.Len(t, g.Events(), 5)
	// While abridged, given we are the creator, we only take 2 events and ignore
	// the member created event
	require.Len(t, g.AbridgedEvents(&creator.PublicKey), 2)
	require.Equal(t, g.AbridgedEvents(&creator.PublicKey)[1].Name, "name-2")

	// While abridged, given we are the new member, we take 4 events
	// that are relevant to us
	require.Len(t, g.AbridgedEvents(&member1.PublicKey), 4)

	// Next is the tricky case, a user that has been invited by someone
	// made an admin. We need to follow the history of admins so
	// that whoever receives the message can see that Creator-> Invited A -> Made A admin -> A Invited B

	// Creator makes member1 Admin
	addedAdminEvent1 := NewAdminsAddedEvent([]string{member1ID}, clock)
	addedAdminEvent1.From = creatorID
	addedAdminEvent1.ChatID = g.chatID
	err = g.ProcessEvent(addedAdminEvent1)
	require.NoError(t, err)
	clock++

	// member1 adds member2
	newMemberEvent2 := NewMembersAddedEvent([]string{member2ID}, clock)
	newMemberEvent2.From = member1ID
	newMemberEvent2.ChatID = g.chatID
	err = g.ProcessEvent(newMemberEvent2)
	require.NoError(t, err)
	clock++

	// member1 makes member2 admin
	addedAdminEvent2 := NewAdminsAddedEvent([]string{member2ID}, clock)
	addedAdminEvent2.From = member1ID
	addedAdminEvent2.ChatID = g.chatID
	err = g.ProcessEvent(addedAdminEvent2)
	require.NoError(t, err)
	clock++

	// member2 adds member3
	newMemberEvent3 := NewMembersAddedEvent([]string{member3ID}, clock)
	newMemberEvent3.From = member2ID
	newMemberEvent3.ChatID = g.chatID
	err = g.ProcessEvent(newMemberEvent3)
	require.NoError(t, err)
	clock++

	// member1 makes member3 admin
	addedAdminEvent3 := NewAdminsAddedEvent([]string{member3ID}, clock)
	addedAdminEvent3.From = member1ID
	addedAdminEvent3.ChatID = g.chatID
	err = g.ProcessEvent(addedAdminEvent3)
	require.NoError(t, err)
	clock++

	// member3 adds member4
	newMemberEvent4 := NewMembersAddedEvent([]string{member4ID}, clock)
	newMemberEvent4.From = member3ID
	newMemberEvent4.ChatID = g.chatID
	err = g.ProcessEvent(newMemberEvent4)
	require.NoError(t, err)

	// Now we check that the history has been correctly followed
	// Full events is 4 events
	require.Len(t, g.Events(), 11)
	// While abridged, given we are the creator, we only take 2 events and ignore
	// the member created event
	require.Len(t, g.AbridgedEvents(&creator.PublicKey), 2)
	require.Equal(t, g.AbridgedEvents(&creator.PublicKey)[1].Name, "name-2")

	// While abridged, given we are the new member, we take 3 events
	// that are relevant to us
	require.Len(t, g.AbridgedEvents(&member4.PublicKey), 9)

	// We build a group from the abridged events

	group, err := NewGroupWithEvents(g.chatID, g.AbridgedEvents(&member4.PublicKey))
	require.NoError(t, err)

	// Make sure the chatID, name is the same
	require.Equal(t, g.name, group.name)
	require.Equal(t, g.chatID, group.chatID)
	// Make sure that user 4 is a member
	require.True(t, group.IsMember(member4ID))
}
