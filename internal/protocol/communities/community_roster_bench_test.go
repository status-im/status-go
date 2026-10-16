package communities

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

const (
	benchRosterSize    = 5000
	benchGatedChannels = 10
)

func benchRosterCommunity(b *testing.B) (*Community, MemberProfileProvider) {
	owner, err := crypto.GenerateKey()
	if err != nil {
		b.Fatal(err)
	}
	description := &protobuf.CommunityDescription{
		Clock:       1,
		Members:     make(map[string]*protobuf.CommunityMember, benchRosterSize),
		Chats:       map[string]*protobuf.CommunityChat{},
		Permissions: &protobuf.CommunityPermissions{},
	}
	profiles := memberProfilesStub{}
	for i := 0; i < benchRosterSize; i++ {
		key, err := crypto.GenerateKey()
		if err != nil {
			b.Fatal(err)
		}
		pubKey := crypto.PubkeyToHex(&key.PublicKey)
		description.Members[pubKey] = &protobuf.CommunityMember{}
		// About one member in ten is a known contact.
		if i%10 == 0 {
			profiles[pubKey] = MemberProfile{DisplayName: "member", ThumbnailImageClock: 1}
		}
	}
	for i := 0; i < benchGatedChannels; i++ {
		chat := &protobuf.CommunityChat{
			Identity:    &protobuf.ChatIdentity{DisplayName: fmt.Sprintf("gated-%d", i)},
			Permissions: &protobuf.CommunityPermissions{},
			Members:     make(map[string]*protobuf.CommunityMember, benchRosterSize),
		}
		for pubKey := range description.Members {
			chat.Members[pubKey] = &protobuf.CommunityMember{}
		}
		description.Chats[fmt.Sprintf("gated-%d", i)] = chat
	}
	community, err := New(Config{
		MemberIdentity:       owner,
		ID:                   &owner.PublicKey,
		PrivateKey:           owner,
		ControlNode:          &owner.PublicKey,
		ControlDevice:        true,
		CommunityDescription: description,
	}, &TimeSourceStub{}, &DescriptionEncryptorMock{}, nil)
	if err != nil {
		b.Fatal(err)
	}
	description.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission, benchGatedChannels)
	for chatID := range description.Chats {
		description.TokenPermissions["gate-"+chatID] = &protobuf.CommunityTokenPermission{
			Id:            "gate-" + chatID,
			Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
			TokenCriteria: []*protobuf.TokenCriteria{{Symbol: "SNT"}},
			ChatIds:       []string{community.ChatID(chatID)},
		}
	}
	for chatID := range description.Chats {
		if !community.ChannelEncrypted(chatID) {
			b.Fatalf("channel %s is not token-gated", chatID)
		}
	}
	return community, profiles
}

func BenchmarkMembersSnapshot(b *testing.B) {
	community, profiles := benchRosterCommunity(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = community.MembersSnapshot(profiles)
	}
}

func BenchmarkMembersSnapshotJSON(b *testing.B) {
	community, profiles := benchRosterCommunity(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytes, err := json.Marshal(community.MembersSnapshot(profiles))
		if err != nil {
			b.Fatal(err)
		}
		b.SetBytes(int64(len(bytes)))
	}
}

// Community JSON as sent in messages.new / community.found.
func BenchmarkCommunityMarshalJSON(b *testing.B) {
	community, _ := benchRosterCommunity(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytes, err := json.Marshal(community)
		if err != nil {
			b.Fatal(err)
		}
		b.SetBytes(int64(len(bytes)))
	}
}
