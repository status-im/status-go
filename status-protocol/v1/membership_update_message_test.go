package statusproto

import (
	"encoding/hex"
	"strings"
	"testing"
	"unicode"

	"github.com/ethereum/go-ethereum/crypto"
	protocrypto "github.com/status-im/status-protocol-go/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testMembershipUpdateMessageBytes  = []byte(`["~#g5",["072ea460-84d3-53c5-9979-1ca36fb5d1020x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1",["~#list",[["^ ","~:chat-id","072ea460-84d3-53c5-9979-1ca36fb5d1020x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1","~:events",[["^ ","~:type","chat-created","~:name","thathata","~:clock-value",156897373998501],["^ ","^4","members-added","^6",156897373998502,"~:members",["~#set",["0x04aebe2bb01a988abe7d978662f21de7760486119876c680e5a559e38e086a2df6dad41c4e4d9079c03db3bced6cb70fca76afc5650e50ea19b81572046a813534"]]]],"~:signature","7fca3d614cf55bc6cdf9c17fd1e65d1688673322bf1f004c58c78e0927edefea3d1053bf6a9d2e058ae88079f588105dccf2a2f9f330f6035cd47c715ee5950601"]]],null]]`)
	testMembershipUpdateMessageStruct = MembershipUpdateMessage{
		ChatID: "072ea460-84d3-53c5-9979-1ca36fb5d1020x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1",
		Updates: []MembershipUpdate{
			{
				ChatID:    "072ea460-84d3-53c5-9979-1ca36fb5d1020x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1",
				Signature: "7fca3d614cf55bc6cdf9c17fd1e65d1688673322bf1f004c58c78e0927edefea3d1053bf6a9d2e058ae88079f588105dccf2a2f9f330f6035cd47c715ee5950601",
				Events: []MembershipUpdateEvent{
					{
						Type:       MembershipUpdateChatCreated,
						Name:       "thathata",
						ClockValue: 156897373998501,
					},
					{
						Type:       MembershipUpdateMembersAdded,
						Members:    []string{"0x04aebe2bb01a988abe7d978662f21de7760486119876c680e5a559e38e086a2df6dad41c4e4d9079c03db3bced6cb70fca76afc5650e50ea19b81572046a813534"},
						ClockValue: 156897373998502,
					},
				},
			},
		},
		Message: nil,
	}
)

func TestDecodeMembershipUpdateMessage(t *testing.T) {
	val, err := decodeTransitMessage(testMembershipUpdateMessageBytes)
	require.NoError(t, err)
	require.EqualValues(t, testMembershipUpdateMessageStruct, val)
}

func TestEncodeMembershipUpdateMessage(t *testing.T) {
	data, err := EncodeMembershipUpdateMessage(testMembershipUpdateMessageStruct)
	require.NoError(t, err)
	// Decode it back to a struct and compare. Comparing bytes is not an option because,
	// for example, map encoding is non-deterministic.
	val, err := decodeTransitMessage(data)
	require.NoError(t, err)
	require.EqualValues(t, testMembershipUpdateMessageStruct, val)
}

func TestTupleMembershipUpdateEvent(t *testing.T) {
	event1 := testMembershipUpdateMessageStruct.Updates[0].Events[0]
	result1 := tupleMembershipUpdateEvent(event1)
	require.EqualValues(t, [][]interface{}{
		{"clock-value", event1.ClockValue},
		{"name", "thathata"},
		{"type", "chat-created"},
	}, result1)

	event2 := testMembershipUpdateMessageStruct.Updates[0].Events[1]
	result2 := tupleMembershipUpdateEvent(event2)
	require.EqualValues(t, [][]interface{}{
		{"clock-value", event2.ClockValue},
		{"members", event2.Members},
		{"type", "members-added"},
	}, result2)
}

func TestSignMembershipUpdate(t *testing.T) {
	key, err := crypto.HexToECDSA("838fbdd1b670209a258b90af25653a018bc582c44c56e6290a973eebbeb15732")
	require.NoError(t, err)
	update := testMembershipUpdateMessageStruct.Updates[0]
	err = update.Sign(key)
	require.NoError(t, err)
	expected, err := protocrypto.SignStringAsHex(
		strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, `
			[
				[
					[
						["clock-value", 156897373998501],
						["name", "thathata"],
						["type", "chat-created"]
					],
					[
						["clock-value", 156897373998502],
						["members", ["0x04aebe2bb01a988abe7d978662f21de7760486119876c680e5a559e38e086a2df6dad41c4e4d9079c03db3bced6cb70fca76afc5650e50ea19b81572046a813534"]],
						["type", "members-added"]
					]
				],
				"072ea460-84d3-53c5-9979-1ca36fb5d1020x0424a68f89ba5fcd5e0640c1e1f591d561fa4125ca4e2a43592bc4123eca10ce064e522c254bb83079ba404327f6eafc01ec90a1444331fe769d3f3a7f90b0dde1"
			]
		`),
		key,
	)
	require.NoError(t, err)
	require.Equal(t, expected, update.Signature)
}

func TestGroupCreator(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	g, err := NewGroupWithCreator("abc", key)
	require.NoError(t, err)
	creator, err := g.creator()
	require.NoError(t, err)
	require.Equal(t, publicKeyToString(&key.PublicKey), creator)
}

func TestGroupProcessEvent(t *testing.T) {
	createGroup := func(admins, members []string, name string) Group {
		return Group{
			name:    name,
			admins:  newStringSetFromSlice(admins),
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
			Group:  createGroup(nil, nil, ""),
			Result: createGroup([]string{"0xabc"}, []string{"0xabc"}, "some-name"),
			From:   "0xabc",
			Event:  NewChatCreatedEvent("some-name", "0xabc", 0),
		},
		{
			Name:   "name-changed event",
			Group:  createGroup(nil, nil, ""),
			Result: createGroup(nil, nil, "some-name"),
			From:   "0xabc",
			Event:  NewNameChangedEvent("some-name", 0),
		},
		{
			Name:   "admins-added event",
			Group:  createGroup(nil, nil, ""),
			Result: createGroup([]string{"0xabc", "0x123"}, nil, ""),
			From:   "0xabc",
			Event:  NewAdminsAddedEvent([]string{"0xabc", "0x123"}, 0),
		},
		{
			Name:   "admin-removed event",
			Group:  createGroup([]string{"0xabc", "0xdef"}, nil, ""),
			Result: createGroup([]string{"0xdef"}, nil, ""),
			From:   "0xabc",
			Event:  NewAdminRemovedEvent("0xabc", 0),
		},
		{
			Name:   "members-added event",
			Group:  createGroup(nil, nil, ""),
			Result: createGroup(nil, []string{"0xabc", "0xdef"}, ""),
			From:   "0xabc",
			Event:  NewMembersAddedEvent([]string{"0xabc", "0xdef"}, 0),
		},
		{
			Name:   "member-removed event",
			Group:  createGroup(nil, []string{"0xabc", "0xdef"}, ""),
			Result: createGroup(nil, []string{"0xdef"}, ""),
			From:   "0xabc",
			Event:  NewMemberRemovedEvent("0xabc", 0),
		},
		{
			Name:   "member-joined event",
			Group:  createGroup(nil, []string{"0xabc"}, ""),
			Result: createGroup(nil, []string{"0xabc", "0xdef"}, ""),
			From:   "0xabc",
			Event:  NewMemberJoinedEvent("0xdef", 0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			g := tc.Group
			g.processEvent(tc.From, tc.Event)
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
			Event:  NewChatCreatedEvent("test", "0xabc", 0),
			Result: true,
		},
		{
			Name:   "chat-created with existing admins",
			Group:  createGroup([]string{"0xabc"}, nil),
			Event:  NewChatCreatedEvent("test", "0xabc", 0),
			Result: false,
		},
		{
			Name:   "chat-created with existing members",
			Group:  createGroup(nil, []string{"0xabc"}),
			Event:  NewChatCreatedEvent("test", "0xabc", 0),
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
			Event:  NewMemberJoinedEvent("0xabc", 0),
			Result: true,
		},
		{
			Name:   "member-joined not valid because not in members",
			From:   "0xabc",
			Group:  createGroup(nil, nil),
			Event:  NewMemberJoinedEvent("0xabc", 0),
			Result: false,
		},
		{
			Name:   "member-joined not valid because from differs from the event",
			From:   "0xdef",
			Group:  createGroup(nil, nil),
			Event:  NewMemberJoinedEvent("0xabc", 0),
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
			result := tc.Group.validateEvent(tc.From, tc.Event)
			assert.Equal(t, tc.Result, result)
		})
	}
}

func TestMembershipUpdateMessageProcess(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	updates := []MembershipUpdate{
		{
			ChatID: "some-chat",
			Events: []MembershipUpdateEvent{
				NewChatCreatedEvent("some-name", "0xabc", 0),
			},
		},
	}
	err = updates[0].Sign(key)
	require.NoError(t, err)
	require.NotEmpty(t, updates[0].Signature)

	message := MembershipUpdateMessage{
		ChatID:  "some-chat",
		Updates: updates,
	}
	err = message.Verify()
	require.NoError(t, err)
	require.EqualValues(t, "0x"+hex.EncodeToString(crypto.FromECDSAPub(&key.PublicKey)), updates[0].From)
}

func TestMembershipUpdateEventEqual(t *testing.T) {
	u1 := MembershipUpdateEvent{
		Type:       MembershipUpdateChatCreated,
		ClockValue: 1,
		Member:     "0xabc",
		Members:    []string{"0xabc"},
		Name:       "abc",
	}
	require.True(t, u1.Equal(u1))

	// Verify equality breaking.
	u2 := u1
	u2.Members = append(u2.Members, "0xdef")
	require.False(t, u1.Equal(u2))
	u2 = u1
	u2.Type = MembershipUpdateMembersAdded
	require.False(t, u1.Equal(u2))
	u2 = u1
	u2.ClockValue = 2
	require.False(t, u1.Equal(u2))
	u2 = u1
	u2.Member = "0xdef"
	require.False(t, u1.Equal(u2))
	u2 = u1
	u2.Name = "def"
	require.False(t, u1.Equal(u2))
}

func TestMembershipUpdateFlatEqual(t *testing.T) {
	u1 := MembershipUpdateFlat{
		ChatID:    "abc",
		Signature: "abc",
		From:      "0xabc",
	}
	require.True(t, u1.Equal(u1))

	// Verify equality breaking.
	u2 := u1
	u2.ChatID = "def"
	require.False(t, u1.Equal(u2))
	u2 = u1
	u2.Signature = "def"
	require.False(t, u1.Equal(u2))
	u2 = u1
	u2.From = "0xdef"
	require.False(t, u1.Equal(u2))
}

func TestMergeFlatMembershipUpdates(t *testing.T) {
	u1 := []MembershipUpdateFlat{
		{
			ChatID:    "abc",
			Signature: "abc",
			From:      "0xabc",
		},
	}
	u2 := []MembershipUpdateFlat{
		{
			ChatID:    "abc",
			Signature: "def",
			From:      "0xdef",
		},
	}
	result := MergeFlatMembershipUpdates(u1, u1)
	require.EqualValues(t, u1, result)
	result = MergeFlatMembershipUpdates(u1, u2)
	require.EqualValues(t, append(u1, u2...), result)
}
