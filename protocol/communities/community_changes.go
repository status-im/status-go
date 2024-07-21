package communities

import (
	"crypto/ecdsa"

	slices "golang.org/x/exp/slices"

	"github.com/status-im/status-go/protocol/protobuf"
)

type CommunityChatChanges struct {
	ChatModified                  *protobuf.CommunityChat
	MembersAdded                  map[string]*protobuf.CommunityMember
	MembersRemoved                map[string]*protobuf.CommunityMember
	CategoryModified              string
	PositionModified              int
	FirstMessageTimestampModified uint32
}

type CommunityChanges struct {
	Community *Community `json:"community"`

	ControlNodeChanged *ecdsa.PublicKey `json:"controlNodeChanged"`

	MembersAdded    map[string]*protobuf.CommunityMember `json:"membersAdded"`
	MembersRemoved  map[string]*protobuf.CommunityMember `json:"membersRemoved"`
	MembersBanned   map[string]bool                      `json:"membersBanned"`
	MembersUnbanned map[string]bool                      `json:"membersUnbanned"`

	TokenPermissions TokenPermissionChanges `json:"tokenPermissions"`

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
}

type TokenPermissionChanges struct {
	Added    TokenPermissions `json:"added"`
	Modified TokenPermissions `json:"modified"`
	Removed  TokenPermissions `json:"removed"`
}

func NewTokenPermissionChanges() TokenPermissionChanges {
	return TokenPermissionChanges{
		Added:    TokenPermissions{},
		Modified: TokenPermissions{},
		Removed:  TokenPermissions{},
	}
}

func EmptyCommunityChanges() *CommunityChanges {
	return &CommunityChanges{
		MembersAdded:    make(map[string]*protobuf.CommunityMember),
		MembersRemoved:  make(map[string]*protobuf.CommunityMember),
		MembersBanned:   make(map[string]bool),
		MembersUnbanned: make(map[string]bool),

		TokenPermissions: TokenPermissionChanges{
			Added:    TokenPermissions{},
			Modified: TokenPermissions{},
			Removed:  TokenPermissions{},
		},

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

func (c *CommunityChanges) Merge(other *CommunityChanges) {
	for memberID, member := range other.MembersAdded {
		c.MembersAdded[memberID] = member
	}
	for memberID := range other.MembersRemoved {
		c.MembersRemoved[memberID] = other.MembersRemoved[memberID]
	}
	for memberID, banned := range other.MembersBanned {
		c.MembersBanned[memberID] = banned
	}
	for memberID, unbanned := range other.MembersUnbanned {
		c.MembersUnbanned[memberID] = unbanned
	}
	for permissionID, permission := range other.TokenPermissions.Added {
		c.TokenPermissions.Added[permissionID] = permission
	}
	for permissionID, permission := range other.TokenPermissions.Modified {
		c.TokenPermissions.Modified[permissionID] = permission
	}
	for permissionID, permission := range other.TokenPermissions.Removed {
		c.TokenPermissions.Removed[permissionID] = permission
	}

	for chatID, chat := range other.ChatsRemoved {
		c.ChatsRemoved[chatID] = chat
	}
	for chatID, chat := range other.ChatsAdded {
		c.ChatsAdded[chatID] = chat
	}
	for chatID, changes := range other.ChatsModified {
		c.ChatsModified[chatID] = changes
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

func EvaluateCommunityChanges(origin, modified *Community) *CommunityChanges {
	changes := evaluateCommunityChangesByDescription(origin.Description(), modified.Description())

	if origin.ControlNode() != nil && !modified.ControlNode().Equal(origin.ControlNode()) {
		changes.ControlNodeChanged = modified.ControlNode()
	}

	changes.TokenPermissions = evaluatePermissionsChanges(origin.tokenPermissions(), modified.tokenPermissions())
	changes.Community = modified

	return changes
}

func evaluateCommunityChangesByDescription(origin, modified *protobuf.CommunityDescription) *CommunityChanges {
	changes := EmptyCommunityChanges()

	// Check for new members at the org level
	for pk, member := range modified.Members {
		if _, ok := origin.Members[pk]; !ok {
			changes.MembersAdded[pk] = member
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

			// Check for members added
			for pk, member := range modified.Chats[chatID].Members {
				if _, ok := origin.Chats[chatID].Members[pk]; !ok {
					if changes.ChatsModified[chatID] == nil {
						changes.ChatsModified[chatID] = &CommunityChatChanges{
							MembersAdded:   make(map[string]*protobuf.CommunityMember),
							MembersRemoved: make(map[string]*protobuf.CommunityMember),
						}
					}
					changes.ChatsModified[chatID].MembersAdded[pk] = member
				}
			}

			// check for members removed
			for pk, member := range origin.Chats[chatID].Members {
				if _, ok := modified.Chats[chatID].Members[pk]; !ok {
					if changes.ChatsModified[chatID] == nil {
						changes.ChatsModified[chatID] = &CommunityChatChanges{
							MembersAdded:   make(map[string]*protobuf.CommunityMember),
							MembersRemoved: make(map[string]*protobuf.CommunityMember),
						}
					}
					changes.ChatsModified[chatID].MembersRemoved[pk] = member
				}
			}

			// check if first message timestamp was modified
			if origin.Chats[chatID].Identity.FirstMessageTimestamp !=
				modified.Chats[chatID].Identity.FirstMessageTimestamp {
				if changes.ChatsModified[chatID] == nil {
					changes.ChatsModified[chatID] = &CommunityChatChanges{
						MembersAdded:   make(map[string]*protobuf.CommunityMember),
						MembersRemoved: make(map[string]*protobuf.CommunityMember),
					}
				}
				changes.ChatsModified[chatID].FirstMessageTimestampModified = modified.Chats[chatID].Identity.FirstMessageTimestamp
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
			if changes.ChatsModified[chatID] == nil {
				changes.ChatsModified[chatID] = &CommunityChatChanges{
					MembersAdded:   make(map[string]*protobuf.CommunityMember),
					MembersRemoved: make(map[string]*protobuf.CommunityMember),
				}
			}

			changes.ChatsModified[chatID].CategoryModified = chat.CategoryId
		}
	}

	return changes
}

func evaluatePermissionsChanges(origin, modified TokenPermissions) TokenPermissionChanges {
	result := TokenPermissionChanges{
		Added:    TokenPermissions{},
		Modified: TokenPermissions{},
		Removed:  TokenPermissions{},
	}

	for id, originPermission := range origin {
		if modifiedPermission := modified[id]; modifiedPermission != nil {
			if !modifiedPermission.Equals(originPermission) {
				result.Modified[id] = modifiedPermission
			}
		} else {
			result.Removed[id] = originPermission
		}
	}

	for id, permission := range modified {
		if _, ok := origin[id]; !ok {
			result.Added[id] = permission
		}
	}

	return result
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
