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
	g, err := NewGroupWithCreator("abc", "#fa6565", 20, key)
	require.NoError(t, err)
	creator, err := g.Creator()
	require.NoError(t, err)
	require.Equal(t, publicKeyToString(&key.PublicKey), creator)
}

func TestGroupProcessEvent(t *testing.T) {
	createGroup := func(admins, members, joined []string, name string, color string, image string) Group {
		return Group{
			name:    name,
			color:   color,
			image:   []byte(image),
			admins:  newStringSetFromSlice(admins),
			members: newStringSetFromSlice(members),
		}
	}

	const emptyName = ""
	const emptyColor = ""
	const emptyImage = ""

	testCases := []struct {
		Name   string
		Group  Group
		Result Group
		From   string
		Event  MembershipUpdateEvent
	}{
		{
			Name:   "chat-created event",
			Group:  createGroup(nil, nil, nil, emptyName, emptyColor, emptyImage),
			Result: createGroup([]string{"0xabc"}, []string{"0xabc"}, []string{"0xabc"}, "some-name", "#7cda00", emptyImage),
			From:   "0xabc",
			Event:  NewChatCreatedEvent("some-name", "#7cda00", 0),
		},
		{
			Name:   "name-changed event",
			Group:  createGroup(nil, nil, nil, emptyName, emptyColor, emptyImage),
			Result: createGroup(nil, nil, nil, "some-name", emptyColor, emptyImage),
			From:   "0xabc",
			Event:  NewNameChangedEvent("some-name", 0),
		},
		{
			Name:   "color-changed event",
			Group:  createGroup(nil, nil, nil, emptyName, emptyColor, emptyImage),
			Result: createGroup(nil, nil, nil, emptyName, "#7cda00", emptyImage),
			From:   "0xabc",
			Event:  NewColorChangedEvent("#7cda00", 0),
		},
		{
			Name:   "image-changed event",
			Group:  createGroup(nil, nil, nil, emptyName, emptyColor, emptyImage),
			Result: createGroup(nil, nil, nil, emptyName, emptyColor, "123"),
			From:   "0xabc",
			Event:  NewImageChangedEvent([]byte("123"), 0),
		},
		{
			Name:   "admins-added event",
			Group:  createGroup(nil, nil, nil, emptyName, emptyColor, emptyImage),
			Result: createGroup([]string{"0xabc", "0x123"}, nil, nil, emptyName, emptyColor, emptyImage),
			From:   "0xabc",
			Event:  NewAdminsAddedEvent([]string{"0xabc", "0x123"}, 0),
		},
		{
			Name:   "admin-removed event",
			Group:  createGroup([]string{"0xabc", "0xdef"}, nil, nil, emptyName, emptyColor, emptyImage),
			Result: createGroup([]string{"0xdef"}, nil, nil, emptyName, emptyColor, emptyImage),
			From:   "0xabc",
			Event:  NewAdminRemovedEvent("0xabc", 0),
		},
		{
			Name:   "members-added event",
			Group:  createGroup(nil, nil, nil, emptyName, emptyColor, emptyImage),
			Result: createGroup(nil, []string{"0xabc", "0xdef"}, nil, emptyName, emptyColor, emptyImage),
			From:   "0xabc",
			Event:  NewMembersAddedEvent([]string{"0xabc", "0xdef"}, 0),
		},
		{
			Name:   "member-removed event",
			Group:  createGroup(nil, []string{"0xabc", "0xdef"}, []string{"0xdef", "0xabc"}, emptyName, emptyColor, emptyImage),
			Result: createGroup(nil, []string{"0xdef"}, []string{"0xdef"}, emptyName, emptyColor, emptyImage),
			From:   "0xabc",
			Event:  NewMemberRemovedEvent([]string{"0xabc"}, 0),
		},
		{
			Name:   "member-joined event",
			Group:  createGroup(nil, []string{"0xabc", "0xdef"}, []string{"0xabc"}, emptyName, emptyColor, emptyImage),
			Result: createGroup(nil, []string{"0xabc", "0xdef"}, []string{"0xabc", "0xdef"}, emptyName, emptyColor, emptyImage),
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
			Event:  NewChatCreatedEvent("test", "#fa6565", 0),
			Result: true,
		},
		{
			Name:   "chat-created with existing admins",
			Group:  createGroup([]string{"0xabc"}, nil),
			From:   "0xabc",
			Event:  NewChatCreatedEvent("test", "#fa6565", 0),
			Result: false,
		},
		{
			Name:   "chat-created with existing members",
			Group:  createGroup(nil, []string{"0xabc"}),
			From:   "0xabc",
			Event:  NewChatCreatedEvent("test", "#fa6565", 0),
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
			Name:   "name-changed allowed because from is member",
			From:   "0x123",
			Group:  createGroup([]string{"0xabc"}, []string{"0x123"}),
			Event:  NewNameChangedEvent("new-name", 0),
			Result: true,
		},
		{
			Name:   "color-changed allowed because from is admin",
			From:   "0xabc",
			Group:  createGroup([]string{"0xabc"}, nil),
			Event:  NewColorChangedEvent("#7cda00", 0),
			Result: true,
		},
		{
			Name:   "color-changed allowed because from is member",
			From:   "0x123",
			Group:  createGroup([]string{"0xabc"}, []string{"0x123"}),
			Event:  NewColorChangedEvent("#7cda00", 0),
			Result: true,
		},
		{
			Name:   "image-changed allowed because from is admin",
			From:   "0xabc",
			Group:  createGroup([]string{"0xabc"}, nil),
			Event:  NewImageChangedEvent([]byte{1, 2, 3}, 0),
			Result: true,
		},
		{
			Name:   "image-changed not allowed for non-admins",
			From:   "0xabc",
			Group:  createGroup(nil, nil),
			Event:  NewImageChangedEvent([]byte{1, 2, 3}, 0),
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
			Name:   "members-added not allowed because from is member",
			From:   "0x123",
			Group:  createGroup([]string{"0xabc"}, []string{"0x123"}),
			Event:  NewMembersAddedEvent([]string{"0x123"}, 0),
			Result: true,
		},
		{
			Name:   "member-removed allowed because removing themselves",
			From:   "0xabc",
			Group:  createGroup(nil, nil),
			Event:  NewMemberRemovedEvent([]string{"0xabc"}, 0),
			Result: true,
		},
		{
			Name:   "member-removed allowed because from is admin",
			From:   "0xabc",
			Group:  createGroup([]string{"0xabc"}, []string{"0x123"}),
			Event:  NewMemberRemovedEvent([]string{"0x123"}, 0),
			Result: true,
		},
		{
			Name:   "member-removed not allowed for non-admins",
			From:   "0x123",
			Group:  createGroup([]string{"0xabc"}, []string{"0x123", "0x456"}),
			Event:  NewMemberRemovedEvent([]string{"0x456"}, 0),
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

func TestAbridgedEventsNameChanged(t *testing.T) {
	var clock uint64 = 0
	creator, err := crypto.GenerateKey()
	require.NoError(t, err)
	creatorID := publicKeyToString(&creator.PublicKey)

	g, err := NewGroupWithCreator("name-0", "#fa6565", clock, creator)
	require.NoError(t, err)
	clock++

	// Full events is only a single one
	require.Len(t, g.Events(), 1)
	// same as abridged
	require.Len(t, g.AbridgedEvents(), 1)

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

	// Full events is 3 events
	require.Len(t, g.Events(), 3)
	// While abridged should exclude the first name-1 event
	require.Len(t, g.AbridgedEvents(), 2)
	require.Equal(t, g.AbridgedEvents()[1].Name, "name-2")
}

func TestAbridgedEventsMembers(t *testing.T) {
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

	g, err := NewGroupWithCreator("name-0", "#fa6565", clock, creator)
	require.NoError(t, err)
	clock++

	// Full events is only a single one
	require.Len(t, g.Events(), 1)
	// same as abridged
	require.Len(t, g.AbridgedEvents(), 1)

	// Add three new members
	event := NewMembersAddedEvent([]string{member1ID, member2ID, member3ID}, clock)
	event.From = creatorID
	event.ChatID = g.chatID
	err = g.ProcessEvent(event)
	require.NoError(t, err)
	clock++

	require.Len(t, g.Events(), 2)
	// All the events are relevant here, so it should be the same
	require.Len(t, g.AbridgedEvents(), 2)

	// We remove one of the users
	event = NewMemberRemovedEvent([]string{member3ID}, clock)
	event.From = creatorID
	event.ChatID = g.chatID
	err = g.ProcessEvent(event)
	require.NoError(t, err)
	clock++

	require.Len(t, g.Events(), 3)
	// All the events are relevant here, so it should be the same
	require.Len(t, g.AbridgedEvents(), 3)

	// Add a new member
	event = NewMembersAddedEvent([]string{member4ID}, clock)
	event.From = creatorID
	event.ChatID = g.chatID
	err = g.ProcessEvent(event)
	require.NoError(t, err)
	clock++

	require.Len(t, g.Events(), 4)
	// All the events are relevant here, so it should be the same
	require.Len(t, g.AbridgedEvents(), 4)

	// We remove the member just added
	event = NewMemberRemovedEvent([]string{member4ID}, clock)
	event.From = creatorID
	event.ChatID = g.chatID
	err = g.ProcessEvent(event)
	require.NoError(t, err)

	require.Len(t, g.Events(), 5)
	// The previous two events, should be removed, because they have no impact
	// on the chat history
	abridgedEvents := g.AbridgedEvents()
	require.Len(t, abridgedEvents, 3)

	require.Equal(t, uint64(0), abridgedEvents[0].ClockValue)
	require.Equal(t, uint64(1), abridgedEvents[1].ClockValue)
	require.Equal(t, uint64(2), abridgedEvents[2].ClockValue)
}

func TestAbridgedEventsAdmins(t *testing.T) {
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

	g, err := NewGroupWithCreator("name-0", "#fa6565", clock, creator)
	require.NoError(t, err)
	clock++

	// Full events is only a single one
	require.Len(t, g.Events(), 1)
	// same as abridged
	require.Len(t, g.AbridgedEvents(), 1)

	// Add three new members
	event := NewMembersAddedEvent([]string{member1ID, member2ID, member3ID}, clock)
	event.From = creatorID
	event.ChatID = g.chatID
	err = g.ProcessEvent(event)
	require.NoError(t, err)
	clock++

	require.Len(t, g.Events(), 2)
	// All the events are relevant here, so it should be the same
	require.Len(t, g.AbridgedEvents(), 2)

	// Make two of them admins
	event = NewAdminsAddedEvent([]string{member1ID, member2ID}, clock)
	event.From = creatorID
	event.ChatID = g.chatID
	err = g.ProcessEvent(event)
	require.NoError(t, err)

	require.Len(t, g.Events(), 3)
	// All the events are relevant here, so it should be the same
	require.Len(t, g.AbridgedEvents(), 3)
}
