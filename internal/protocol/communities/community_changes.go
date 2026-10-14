package communities

import (
	"bytes"
	"cmp"
	"crypto/ecdsa"
	"encoding/json"
	"maps"
	"slices"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

type CommunityChatChanges struct {
	ChatModified                  *protobuf.CommunityChat
	MembersAdded                  map[string]*protobuf.CommunityMember
	MembersRemoved                map[string]*protobuf.CommunityMember
	MembersModified               map[string]*protobuf.CommunityMember // channel role changed
	CategoryModified              string
	PositionModified              int
	FirstMessageTimestampModified uint32
}

type CommunityChanges struct {
	// Sent once, in the response's communities, not again with its changes.
	Community *Community `json:"-"`

	ControlNodeChanged *ecdsa.PublicKey `json:"controlNodeChanged"`

	MembersAdded    map[string]*protobuf.CommunityMember `json:"membersAdded"`
	MembersRemoved  map[string]*protobuf.CommunityMember `json:"membersRemoved"`
	MembersModified map[string]*protobuf.CommunityMember `json:"membersModified"` // roles changed
	MembersBanned   map[string]bool                      `json:"membersBanned"`
	MembersUnbanned map[string]bool                      `json:"membersUnbanned"`

	TokenPermissionsAdded    map[string]*CommunityTokenPermission `json:"tokenPermissionsAdded"`
	TokenPermissionsModified map[string]*CommunityTokenPermission `json:"tokenPermissionsModified"`
	TokenPermissionsRemoved  map[string]*CommunityTokenPermission `json:"tokenPermissionsRemoved"`

	ChatsRemoved  map[string]*protobuf.CommunityChat `json:"chatsRemoved"`
	ChatsAdded    map[string]*protobuf.CommunityChat `json:"chatsAdded"`
	ChatsModified map[string]*CommunityChatChanges   `json:"chatsModified"`

	CategoriesRemoved  []string                               `json:"categoriesRemoved"`
	CategoriesAdded    map[string]*protobuf.CommunityCategory `json:"categoriesAdded"`
	CategoriesModified map[string]*protobuf.CommunityCategory `json:"categoriesModified"`

	MemberWalletsRemoved []string                               `json:"memberWalletsRemoved"`
	MemberWalletsAdded   map[string][]*protobuf.RevealedAccount `json:"memberWalletsAdded"`

	// ShouldMemberJoin indicates whether the user should join this community
	// automatically
	ShouldMemberJoin bool `json:"memberAdded"`

	// MemberKicked indicates whether the user has been kicked out
	MemberKicked bool `json:"memberRemoved"`

	// MemberSoftKicked indicates whether the user has been kicked out due to lack of specific data
	// No kick AC notification will be generated and member will join automatically
	// as soon as he provides missing data
	MemberSoftKicked bool `json:"memberSoftRemoved"`

	// CommunityImageModified indicates whether the community image was modified by an admin or owner
	ImageModified bool `json:"communityImageModified"`
}

func EmptyCommunityChanges() *CommunityChanges {
	return &CommunityChanges{
		MembersAdded:    make(map[string]*protobuf.CommunityMember),
		MembersRemoved:  make(map[string]*protobuf.CommunityMember),
		MembersModified: make(map[string]*protobuf.CommunityMember),
		MembersBanned:   make(map[string]bool),
		MembersUnbanned: make(map[string]bool),

		TokenPermissionsAdded:    make(map[string]*CommunityTokenPermission),
		TokenPermissionsModified: make(map[string]*CommunityTokenPermission),
		TokenPermissionsRemoved:  make(map[string]*CommunityTokenPermission),

		ChatsRemoved:  make(map[string]*protobuf.CommunityChat),
		ChatsAdded:    make(map[string]*protobuf.CommunityChat),
		ChatsModified: make(map[string]*CommunityChatChanges),

		CategoriesRemoved:  []string{},
		CategoriesAdded:    make(map[string]*protobuf.CommunityCategory),
		CategoriesModified: make(map[string]*protobuf.CommunityCategory),

		MemberWalletsRemoved: []string{},
		MemberWalletsAdded:   make(map[string][]*protobuf.RevealedAccount),
	}
}

// Merge applies other on top of c, which must be the earlier of the two.
func (c *CommunityChanges) Merge(other *CommunityChanges) {
	mergeMemberDeltas(&c.MembersAdded, &c.MembersRemoved, &c.MembersModified,
		other.MembersAdded, other.MembersRemoved, other.MembersModified)
	for memberID, banned := range other.MembersBanned {
		c.MembersBanned[memberID] = banned
	}
	for memberID, unbanned := range other.MembersUnbanned {
		c.MembersUnbanned[memberID] = unbanned
	}
	for permissionID, permission := range other.TokenPermissionsAdded {
		c.TokenPermissionsAdded[permissionID] = permission
	}
	for permissionID, permission := range other.TokenPermissionsModified {
		c.TokenPermissionsModified[permissionID] = permission
	}
	for permissionID, permission := range other.TokenPermissionsRemoved {
		c.TokenPermissionsRemoved[permissionID] = permission
	}
	for chatID, chat := range other.ChatsRemoved {
		c.ChatsRemoved[chatID] = chat
	}
	for chatID, chat := range other.ChatsAdded {
		c.ChatsAdded[chatID] = chat
	}
	for chatID, changes := range other.ChatsModified {
		existing, ok := c.ChatsModified[chatID]
		if !ok {
			c.ChatsModified[chatID] = changes
			continue
		}
		merged := *existing
		merged.MembersAdded = maps.Clone(existing.MembersAdded)
		merged.MembersRemoved = maps.Clone(existing.MembersRemoved)
		merged.MembersModified = maps.Clone(existing.MembersModified)
		merged.merge(changes)
		c.ChatsModified[chatID] = &merged
	}

	c.CategoriesRemoved = append(c.CategoriesRemoved, other.CategoriesRemoved...)

	for categoryID, category := range other.CategoriesAdded {
		c.CategoriesAdded[categoryID] = category
	}
	for categoryID, category := range other.CategoriesModified {
		c.CategoriesModified[categoryID] = category
	}

	c.MemberWalletsRemoved = append(c.MemberWalletsRemoved, other.MemberWalletsRemoved...)

	for walletID, wallets := range other.MemberWalletsAdded {
		c.MemberWalletsAdded[walletID] = wallets
	}
}

// MarshalJSON adds the community ID and clock and the member deltas as
// roster entries next to the existing keys.
func (c *CommunityChanges) MarshalJSON() ([]byte, error) {
	type changesAlias CommunityChanges
	item := struct {
		*changesAlias
		CommunityID    types.HexBytes         `json:"communityId,omitempty"`
		Clock          uint64                 `json:"clock"`
		RosterAdded    map[string]RosterEntry `json:"rosterAdded"`
		RosterRemoved  map[string]RosterEntry `json:"rosterRemoved"`
		RosterModified map[string]RosterEntry `json:"rosterModified"`
	}{
		changesAlias: (*changesAlias)(c),
	}

	var profiles MemberProfileProvider
	if c.Community != nil {
		item.CommunityID = c.Community.ID()
		item.Clock = c.Community.latestClock()
		profiles = c.Community.memberProfiles
	}
	item.RosterAdded = rosterEntries(c.MembersAdded, profiles)
	item.RosterRemoved = rosterEntries(c.MembersRemoved, profiles)
	item.RosterModified = rosterEntries(c.MembersModified, profiles)

	return json.Marshal(item)
}

// chatChanges returns the changes of chatID, creating them if needed.
func (c *CommunityChanges) chatChanges(chatID string) *CommunityChatChanges {
	if c.ChatsModified[chatID] == nil {
		c.ChatsModified[chatID] = &CommunityChatChanges{
			MembersAdded:    make(map[string]*protobuf.CommunityMember),
			MembersRemoved:  make(map[string]*protobuf.CommunityMember),
			MembersModified: make(map[string]*protobuf.CommunityMember),
		}
	}
	return c.ChatsModified[chatID]
}

// HasRosterChanges reports whether the changes add, remove or modify a
// community or channel member.
func (c *CommunityChanges) HasRosterChanges() bool {
	if len(c.MembersAdded) > 0 || len(c.MembersRemoved) > 0 || len(c.MembersModified) > 0 {
		return true
	}
	for _, chat := range c.ChatsModified {
		if len(chat.MembersAdded) > 0 || len(chat.MembersRemoved) > 0 || len(chat.MembersModified) > 0 {
			return true
		}
	}
	return false
}

func (c *CommunityChanges) HasNewMember(identity string) bool {
	if len(c.MembersAdded) == 0 {
		return false
	}
	_, ok := c.MembersAdded[identity]
	return ok
}

func (c *CommunityChanges) HasMemberLeft(identity string) bool {
	if len(c.MembersRemoved) == 0 {
		return false
	}
	_, ok := c.MembersRemoved[identity]
	return ok
}

func (c *CommunityChanges) IsMemberBanned(identity string) bool {
	if len(c.MembersBanned) == 0 {
		return false
	}
	_, ok := c.MembersBanned[identity]
	return ok
}

func (c *CommunityChanges) IsMemberUnbanned(identity string) bool {
	if len(c.MembersUnbanned) == 0 {
		return false
	}
	_, ok := c.MembersUnbanned[identity]
	return ok
}

func CommunityImagesChanged(originCommunityImages, modifiedCommunityImages map[string]*protobuf.IdentityImage) bool {
	for imageType, newImage := range modifiedCommunityImages {
		oldImage, ok := originCommunityImages[imageType]
		if ok {
			if !bytes.Equal(oldImage.Payload, newImage.Payload) {
				return true
			}
		}
	}
	return false
}

func EvaluateCommunityChanges(origin, modified *Community) *CommunityChanges {
	changes := evaluateCommunityChangesByDescription(origin.Description(), modified.Description())

	if origin.ControlNode() != nil && !modified.ControlNode().Equal(origin.ControlNode()) {
		changes.ControlNodeChanged = modified.ControlNode()
	}

	originTokenPermissions := origin.tokenPermissions()
	modifiedTokenPermissions := modified.tokenPermissions()

	// Check for modified or removed token permissions
	for id, originPermission := range originTokenPermissions {
		if modifiedPermission := modifiedTokenPermissions[id]; modifiedPermission != nil {
			if !modifiedPermission.Equals(originPermission) {
				changes.TokenPermissionsModified[id] = modifiedPermission
			}
		} else {
			changes.TokenPermissionsRemoved[id] = originPermission
		}
	}

	// Check for added token permissions
	for id, permission := range modifiedTokenPermissions {
		if _, ok := originTokenPermissions[id]; !ok {
			changes.TokenPermissionsAdded[id] = permission
		}
	}

	changes.ImageModified = CommunityImagesChanged(modified.config.CommunityDescription.Identity.Images, origin.config.CommunityDescription.Identity.Images)

	changes.Community = modified
	return changes
}

func evaluateCommunityChangesByDescription(origin, modified *protobuf.CommunityDescription) *CommunityChanges {
	changes := EmptyCommunityChanges()

	// Check for new members and role changes at the org level
	for pk, member := range modified.Members {
		originMember, ok := origin.Members[pk]
		if !ok {
			changes.MembersAdded[pk] = member
		} else if !sameRoles(originMember.GetRoles(), member.GetRoles()) {
			changes.MembersModified[pk] = member
		}
	}

	// Check ban/unban
	findDiffInBannedMembers(modified.BannedMembers, origin.BannedMembers, changes.MembersBanned)
	findDiffInBannedMembers(origin.BannedMembers, modified.BannedMembers, changes.MembersUnbanned)

	// Check for new banned members (from deprecated BanList)
	findDiffInBanList(modified.BanList, origin.BanList, changes.MembersBanned)

	// Check for new unbanned members (from deprecated BanList)
	findDiffInBanList(origin.BanList, modified.BanList, changes.MembersUnbanned)

	// Check for removed members at the org level
	for pk, member := range origin.Members {
		if _, ok := modified.Members[pk]; !ok {
			changes.MembersRemoved[pk] = member
		}
	}

	// check for removed chats
	for chatID, chat := range origin.Chats {
		if modified.Chats == nil {
			modified.Chats = make(map[string]*protobuf.CommunityChat)
		}
		if _, ok := modified.Chats[chatID]; !ok {
			changes.ChatsRemoved[chatID] = chat
		}
	}

	for chatID, chat := range modified.Chats {
		if origin.Chats == nil {
			origin.Chats = make(map[string]*protobuf.CommunityChat)
		}

		if _, ok := origin.Chats[chatID]; !ok {
			changes.ChatsAdded[chatID] = chat
		} else {

			// Check for members added and channel role changes
			for pk, member := range modified.Chats[chatID].Members {
				originMember, ok := origin.Chats[chatID].Members[pk]
				if !ok {
					changes.chatChanges(chatID).MembersAdded[pk] = member
				} else if originMember.GetChannelRole() != member.GetChannelRole() {
					changes.chatChanges(chatID).MembersModified[pk] = member
				}
			}

			// check for members removed
			for pk, member := range origin.Chats[chatID].Members {
				if _, ok := modified.Chats[chatID].Members[pk]; !ok {
					changes.chatChanges(chatID).MembersRemoved[pk] = member
				}
			}

			// check if first message timestamp was modified
			if origin.Chats[chatID].Identity.FirstMessageTimestamp !=
				modified.Chats[chatID].Identity.FirstMessageTimestamp {
				changes.chatChanges(chatID).FirstMessageTimestampModified = modified.Chats[chatID].Identity.FirstMessageTimestamp
			}
		}
	}

	// Check for categories that were removed
	for categoryID := range origin.Categories {
		if modified.Categories == nil {
			modified.Categories = make(map[string]*protobuf.CommunityCategory)
		}

		if modified.Chats == nil {
			modified.Chats = make(map[string]*protobuf.CommunityChat)
		}

		if _, ok := modified.Categories[categoryID]; !ok {
			changes.CategoriesRemoved = append(changes.CategoriesRemoved, categoryID)
		}

		if origin.Chats == nil {
			origin.Chats = make(map[string]*protobuf.CommunityChat)
		}
	}

	// Check for categories that were added
	for categoryID, category := range modified.Categories {
		if origin.Categories == nil {
			origin.Categories = make(map[string]*protobuf.CommunityCategory)
		}
		if _, ok := origin.Categories[categoryID]; !ok {
			changes.CategoriesAdded[categoryID] = category
		} else {
			if origin.Categories[categoryID].Name != category.Name || origin.Categories[categoryID].Position != category.Position {
				changes.CategoriesModified[categoryID] = category
			}
		}
	}

	// Check for chat categories that were modified
	for chatID, chat := range modified.Chats {
		if origin.Chats == nil {
			origin.Chats = make(map[string]*protobuf.CommunityChat)
		}

		if _, ok := origin.Chats[chatID]; !ok {
			continue // It's a new chat
		}

		if origin.Chats[chatID].CategoryId != chat.CategoryId {
			changes.chatChanges(chatID).CategoryModified = chat.CategoryId
		}
	}

	return changes
}

func sameRoles(a, b []protobuf.CommunityMember_Roles) bool {
	if len(a) != len(b) {
		return false
	}
	for _, role := range a {
		if !slices.Contains(b, role) {
			return false
		}
	}
	return true
}

func findDiffInBanList(searchFrom []string, searchIn []string, storeTo map[string]bool) {
	for _, memberToFind := range searchFrom {
		if _, stored := storeTo[memberToFind]; stored {
			continue
		}

		exists := slices.Contains(searchIn, memberToFind)

		if !exists {
			storeTo[memberToFind] = false
		}
	}
}

func findDiffInBannedMembers(searchFrom map[string]*protobuf.CommunityBanInfo, searchIn map[string]*protobuf.CommunityBanInfo, storeTo map[string]bool) {
	if searchFrom == nil {
		return
	} else if searchIn == nil {
		for memberToFind, value := range searchFrom {
			storeTo[memberToFind] = value.DeleteAllMessages
		}
	} else {
		for memberToFind, value := range searchFrom {
			if _, exists := searchIn[memberToFind]; !exists {
				storeTo[memberToFind] = value.DeleteAllMessages
			}
		}
	}
}

func (c *CommunityChatChanges) merge(other *CommunityChatChanges) {
	mergeMemberDeltas(&c.MembersAdded, &c.MembersRemoved, &c.MembersModified,
		other.MembersAdded, other.MembersRemoved, other.MembersModified)
	if other.ChatModified != nil {
		c.ChatModified = other.ChatModified
	}
	if other.CategoryModified != "" {
		c.CategoryModified = other.CategoryModified
	}
	if other.PositionModified != 0 {
		c.PositionModified = other.PositionModified
	}
	if other.FirstMessageTimestampModified != 0 {
		c.FirstMessageTimestampModified = other.FirstMessageTimestampModified
	}
}

// mergeMemberDeltas applies later member deltas on top of earlier ones, so a
// member ends up in at most one of the earlier maps.
func mergeMemberDeltas(added, removed, modified *map[string]*protobuf.CommunityMember,
	laterAdded, laterRemoved, laterModified map[string]*protobuf.CommunityMember) {
	for _, m := range []*map[string]*protobuf.CommunityMember{added, removed, modified} {
		if *m == nil {
			*m = make(map[string]*protobuf.CommunityMember)
		}
	}

	for pk, member := range laterRemoved {
		delete(*added, pk)
		delete(*modified, pk)
		(*removed)[pk] = member
	}
	for pk, member := range laterAdded {
		delete(*removed, pk)
		delete(*modified, pk)
		(*added)[pk] = member
	}
	for pk, member := range laterModified {
		if _, ok := (*added)[pk]; ok {
			(*added)[pk] = member
			continue
		}
		(*modified)[pk] = member
	}
}

// CoalesceCommunityChanges merges the changes of each community into one
// entry, in clock order, keeping the first position of each community.
// Changes without a community are kept as they are.
func CoalesceCommunityChanges(changes []*CommunityChanges) []*CommunityChanges {
	groups := make(map[string][]*CommunityChanges, len(changes))
	order := make([]*CommunityChanges, 0, len(changes))
	for _, change := range changes {
		if change.Community == nil {
			order = append(order, change)
			continue
		}
		id := change.Community.IDString()
		if _, ok := groups[id]; !ok {
			order = append(order, change)
		}
		groups[id] = append(groups[id], change)
	}

	result := make([]*CommunityChanges, 0, len(order))
	for _, change := range order {
		if change.Community == nil {
			result = append(result, change)
			continue
		}
		group := groups[change.Community.IDString()]
		if len(group) == 1 {
			result = append(result, change)
			continue
		}
		slices.SortStableFunc(group, func(a, b *CommunityChanges) int {
			return cmp.Compare(a.Community.latestClock(), b.Community.latestClock())
		})
		merged := EmptyCommunityChanges()
		for _, entry := range group {
			merged.Merge(entry)
		}
		merged.Community = group[len(group)-1].Community
		syncChatPlacement(merged)
		result = append(result, merged)
	}
	return result
}

// syncChatPlacement sets the category and position of each modified chat
// from the merged community, as merging cannot tell a reset to the zero
// value from a change that left them unset.
func syncChatPlacement(changes *CommunityChanges) {
	chats := changes.Community.Chats()
	for chatID, chatChanges := range changes.ChatsModified {
		chat, ok := chats[chatID]
		if !ok {
			continue
		}
		// Merge may share the entry with an input change.
		synced := *chatChanges
		synced.CategoryModified = chat.CategoryId
		synced.PositionModified = int(chat.Position)
		changes.ChatsModified[chatID] = &synced
	}
}
