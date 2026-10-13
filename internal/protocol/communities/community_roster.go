package communities

import (
	"github.com/status-im/status-go/internal/protocol/identity"
	"github.com/status-im/status-go/internal/protocol/identity/alias"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

// MemberProfile is the display data known locally for a community member.
type MemberProfile struct {
	DisplayName         string
	EnsName             string
	EnsVerified         bool
	ThumbnailImageClock uint64
}

// MemberProfileProvider resolves display data for a member's hex public key.
type MemberProfileProvider interface {
	MemberProfile(pubKey string) (MemberProfile, bool)
}

// RosterEntry is the slim per-member record sent to clients in roster
// snapshots and member deltas.
type RosterEntry struct {
	PubKey      string                           `json:"pubKey"`
	Roles       []protobuf.CommunityMember_Roles `json:"roles"`
	DisplayName string                           `json:"displayName,omitempty"`
	EnsName     string                           `json:"ensName,omitempty"`
	EnsVerified bool                             `json:"ensVerified,omitempty"`
	// Alias is only set when the member has no known display name.
	Alias   string `json:"alias,omitempty"`
	ColorID int64  `json:"colorId"`
	// ThumbnailImageClock is the clock of the member's thumbnail image, as
	// used by the media server contact image URL; 0 when unknown.
	ThumbnailImageClock uint64 `json:"thumbnailImageClock,omitempty"`
}

// CommunityMembers is an atomic snapshot of a community roster at Clock.
type CommunityMembers struct {
	Clock   uint64        `json:"clock"`
	Members []RosterEntry `json:"members"`
	// Channels holds the channel role of every member of each token-gated
	// channel, keyed by channel ID and member public key.
	Channels map[string]map[string]protobuf.CommunityMember_ChannelRole `json:"channels"`
}

func newRosterEntry(pubKey string, member *protobuf.CommunityMember, profiles MemberProfileProvider) RosterEntry {
	entry := RosterEntry{
		PubKey: pubKey,
		Roles:  member.GetRoles(),
	}
	if entry.Roles == nil {
		entry.Roles = []protobuf.CommunityMember_Roles{}
	}
	if profiles != nil {
		if profile, ok := profiles.MemberProfile(pubKey); ok {
			entry.DisplayName = profile.DisplayName
			entry.EnsName = profile.EnsName
			entry.EnsVerified = profile.EnsVerified
			entry.ThumbnailImageClock = profile.ThumbnailImageClock
		}
	}
	if entry.DisplayName == "" {
		entry.Alias, _ = alias.GenerateFromPublicKeyString(pubKey)
	}
	entry.ColorID, _ = identity.ToColorID(pubKey)
	return entry
}

func rosterEntries(members map[string]*protobuf.CommunityMember, profiles MemberProfileProvider) map[string]RosterEntry {
	if members == nil {
		return nil
	}
	entries := make(map[string]RosterEntry, len(members))
	for pubKey, member := range members {
		entries[pubKey] = newRosterEntry(pubKey, member, profiles)
	}
	return entries
}

// MembersSnapshot returns the community roster and the channel roles of
// token-gated channels, consistent with the returned clock.
func (o *Community) MembersSnapshot(profiles MemberProfileProvider) *CommunityMembers {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	description := o.config.CommunityDescription
	snapshot := &CommunityMembers{
		Clock:    description.GetClock(),
		Members:  make([]RosterEntry, 0, len(description.GetMembers())),
		Channels: make(map[string]map[string]protobuf.CommunityMember_ChannelRole),
	}
	for pubKey, member := range description.GetMembers() {
		snapshot.Members = append(snapshot.Members, newRosterEntry(pubKey, member, profiles))
	}
	for chatID, chat := range description.GetChats() {
		if !o.channelEncrypted(chatID) {
			continue
		}
		channelRoles := make(map[string]protobuf.CommunityMember_ChannelRole, len(chat.GetMembers()))
		for pubKey, member := range chat.GetMembers() {
			channelRoles[pubKey] = member.GetChannelRole()
		}
		snapshot.Channels[chatID] = channelRoles
	}
	return snapshot
}
