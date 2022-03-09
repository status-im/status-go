package communities

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/v1"
)

const signatureLength = 65

type Config struct {
	PrivateKey                    *ecdsa.PrivateKey
	CommunityDescription          *protobuf.CommunityDescription
	MarshaledCommunityDescription []byte
	ID                            *ecdsa.PublicKey
	Joined                        bool
	Requested                     bool
	Verified                      bool
	Muted                         bool
	Logger                        *zap.Logger
	RequestedToJoinAt             uint64
	RequestsToJoin                []*RequestToJoin
	MemberIdentity                *ecdsa.PublicKey
	SyncedAt                      uint64
}

type Community struct {
	config *Config
	mutex  sync.Mutex
}

func New(config Config) (*Community, error) {
	if config.MemberIdentity == nil {
		return nil, errors.New("no member identity")
	}

	if config.Logger == nil {
		logger, err := zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
		config.Logger = logger
	}

	community := &Community{config: &config}
	community.initialize()
	return community, nil
}

type CommunityChat struct {
	ID          string                               `json:"id"`
	Name        string                               `json:"name"`
	Color       string                               `json:"color"`
	Emoji       string                               `json:"emoji"`
	Description string                               `json:"description"`
	Members     map[string]*protobuf.CommunityMember `json:"members"`
	Permissions *protobuf.CommunityPermissions       `json:"permissions"`
	CanPost     bool                                 `json:"canPost"`
	Position    int                                  `json:"position"`
	CategoryID  string                               `json:"categoryID"`
}

type CommunityCategory struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"` // Position is used to sort the categories
}

func (o *Community) MarshalPublicAPIJSON() ([]byte, error) {
	if o.config.MemberIdentity == nil {
		return nil, errors.New("member identity not set")
	}
	communityItem := struct {
		ID           types.HexBytes                  `json:"id"`
		Verified     bool                            `json:"verified"`
		Chats        map[string]CommunityChat        `json:"chats"`
		Categories   map[string]CommunityCategory    `json:"categories"`
		Name         string                          `json:"name"`
		Description  string                          `json:"description"`
		Images       map[string]images.IdentityImage `json:"images"`
		Color        string                          `json:"color"`
		MembersCount int                             `json:"membersCount"`
		EnsName      string                          `json:"ensName"`
		Link         string                          `json:"link"`
	}{
		ID:         o.ID(),
		Verified:   o.config.Verified,
		Chats:      make(map[string]CommunityChat),
		Categories: make(map[string]CommunityCategory),
	}
	if o.config.CommunityDescription != nil {
		for id, c := range o.config.CommunityDescription.Categories {
			category := CommunityCategory{
				ID:       id,
				Name:     c.Name,
				Position: int(c.Position),
			}
			communityItem.Categories[id] = category
		}
		for id, c := range o.config.CommunityDescription.Chats {
			canPost, err := o.CanPost(o.config.MemberIdentity, id, nil)
			if err != nil {
				return nil, err
			}
			chat := CommunityChat{
				ID:          id,
				Name:        c.Identity.DisplayName,
				Color:       c.Identity.Color,
				Emoji:       c.Identity.Emoji,
				Description: c.Identity.Description,
				Permissions: c.Permissions,
				Members:     c.Members,
				CanPost:     canPost,
				CategoryID:  c.CategoryId,
				Position:    int(c.Position),
			}
			communityItem.Chats[id] = chat
		}
		communityItem.MembersCount = len(o.config.CommunityDescription.Members)
		communityItem.Link = fmt.Sprintf("https://join.status.im/c/0x%x", o.ID())
		if o.config.CommunityDescription.Identity != nil {
			communityItem.Name = o.Name()
			communityItem.Color = o.config.CommunityDescription.Identity.Color
			communityItem.Description = o.config.CommunityDescription.Identity.Description
			for t, i := range o.config.CommunityDescription.Identity.Images {
				if communityItem.Images == nil {
					communityItem.Images = make(map[string]images.IdentityImage)
				}
				communityItem.Images[t] = images.IdentityImage{Name: t, Payload: i.Payload}

			}
		}

	}
	return json.Marshal(communityItem)
}

func (o *Community) MarshalJSON() ([]byte, error) {
	if o.config.MemberIdentity == nil {
		return nil, errors.New("member identity not set")
	}
	communityItem := struct {
		ID                types.HexBytes                       `json:"id"`
		Admin             bool                                 `json:"admin"`
		Verified          bool                                 `json:"verified"`
		Joined            bool                                 `json:"joined"`
		RequestedAccessAt int                                  `json:"requestedAccessAt"`
		Name              string                               `json:"name"`
		Description       string                               `json:"description"`
		Chats             map[string]CommunityChat             `json:"chats"`
		Categories        map[string]CommunityCategory         `json:"categories"`
		Images            map[string]images.IdentityImage      `json:"images"`
		Permissions       *protobuf.CommunityPermissions       `json:"permissions"`
		Members           map[string]*protobuf.CommunityMember `json:"members"`
		CanRequestAccess  bool                                 `json:"canRequestAccess"`
		CanManageUsers    bool                                 `json:"canManageUsers"`
		CanJoin           bool                                 `json:"canJoin"`
		Color             string                               `json:"color"`
		RequestedToJoinAt uint64                               `json:"requestedToJoinAt,omitempty"`
		IsMember          bool                                 `json:"isMember"`
		Muted             bool                                 `json:"muted"`
	}{
		ID:                o.ID(),
		Admin:             o.IsAdmin(),
		Verified:          o.config.Verified,
		Chats:             make(map[string]CommunityChat),
		Categories:        make(map[string]CommunityCategory),
		Joined:            o.config.Joined,
		CanRequestAccess:  o.CanRequestAccess(o.config.MemberIdentity),
		CanJoin:           o.canJoin(),
		CanManageUsers:    o.CanManageUsers(o.config.MemberIdentity),
		RequestedToJoinAt: o.RequestedToJoinAt(),
		IsMember:          o.isMember(),
		Muted:             o.config.Muted,
	}
	if o.config.CommunityDescription != nil {
		for id, c := range o.config.CommunityDescription.Categories {
			category := CommunityCategory{
				ID:       id,
				Name:     c.Name,
				Position: int(c.Position),
			}
			communityItem.Categories[id] = category
		}
		for id, c := range o.config.CommunityDescription.Chats {
			canPost, err := o.CanPost(o.config.MemberIdentity, id, nil)
			if err != nil {
				return nil, err
			}
			chat := CommunityChat{
				ID:          id,
				Name:        c.Identity.DisplayName,
				Emoji:       c.Identity.Emoji,
				Color:       c.Identity.Color,
				Description: c.Identity.Description,
				Permissions: c.Permissions,
				Members:     c.Members,
				CanPost:     canPost,
				CategoryID:  c.CategoryId,
				Position:    int(c.Position),
			}
			communityItem.Chats[id] = chat
		}
		communityItem.Members = o.config.CommunityDescription.Members
		communityItem.Permissions = o.config.CommunityDescription.Permissions
		if o.config.CommunityDescription.Identity != nil {
			communityItem.Name = o.Name()
			communityItem.Color = o.config.CommunityDescription.Identity.Color
			communityItem.Description = o.config.CommunityDescription.Identity.Description
			for t, i := range o.config.CommunityDescription.Identity.Images {
				if communityItem.Images == nil {
					communityItem.Images = make(map[string]images.IdentityImage)
				}
				communityItem.Images[t] = images.IdentityImage{Name: t, Payload: i.Payload}

			}
		}

	}
	return json.Marshal(communityItem)
}

func (o *Community) Name() string {
	if o != nil &&
		o.config != nil &&
		o.config.CommunityDescription != nil &&
		o.config.CommunityDescription.Identity != nil {
		return o.config.CommunityDescription.Identity.DisplayName
	}
	return ""
}

func (o *Community) DescriptionText() string {
	if o != nil &&
		o.config != nil &&
		o.config.CommunityDescription != nil &&
		o.config.CommunityDescription.Identity != nil {
		return o.config.CommunityDescription.Identity.Description
	}
	return ""
}

func (o *Community) MembersCount() int {
	if o != nil &&
		o.config != nil &&
		o.config.CommunityDescription != nil {
		return len(o.config.CommunityDescription.Members)
	}
	return 0
}

func (o *Community) initialize() {
	if o.config.CommunityDescription == nil {
		o.config.CommunityDescription = &protobuf.CommunityDescription{}

	}
}

type CommunitySettings struct {
	CommunityID                  string `json:"communityId"`
	HistoryArchiveSupportEnabled bool   `json:"historyArchiveSupportEnabled"`
}

type CommunityChatChanges struct {
	ChatModified     *protobuf.CommunityChat
	MembersAdded     map[string]*protobuf.CommunityMember
	MembersRemoved   map[string]*protobuf.CommunityMember
	CategoryModified string
	PositionModified int
}

type CommunityChanges struct {
	Community      *Community                           `json:"community"`
	MembersAdded   map[string]*protobuf.CommunityMember `json:"membersAdded"`
	MembersRemoved map[string]*protobuf.CommunityMember `json:"membersRemoved"`

	ChatsRemoved  map[string]*protobuf.CommunityChat `json:"chatsRemoved"`
	ChatsAdded    map[string]*protobuf.CommunityChat `json:"chatsAdded"`
	ChatsModified map[string]*CommunityChatChanges   `json:"chatsModified"`

	CategoriesRemoved  []string                               `json:"categoriesRemoved"`
	CategoriesAdded    map[string]*protobuf.CommunityCategory `json:"categoriesAdded"`
	CategoriesModified map[string]*protobuf.CommunityCategory `json:"categoriesModified"`

	// ShouldMemberJoin indicates whether the user should join this community
	// automatically
	ShouldMemberJoin bool `json:"memberAdded"`

	// ShouldMemberJoin indicates whether the user should leave this community
	// automatically
	ShouldMemberLeave bool `json:"memberRemoved"`
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

func (o *Community) emptyCommunityChanges() *CommunityChanges {
	changes := emptyCommunityChanges()
	changes.Community = o
	return changes
}

func (o *Community) CreateChat(chatID string, chat *protobuf.CommunityChat) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.IsAdmin() {
		return nil, ErrNotAdmin
	}

	err := validateCommunityChat(o.config.CommunityDescription, chat)
	if err != nil {
		return nil, err
	}

	if o.config.CommunityDescription.Chats == nil {
		o.config.CommunityDescription.Chats = make(map[string]*protobuf.CommunityChat)
	}
	if _, ok := o.config.CommunityDescription.Chats[chatID]; ok {
		return nil, ErrChatAlreadyExists
	}

	for _, c := range o.config.CommunityDescription.Chats {
		if chat.Identity.DisplayName == c.Identity.DisplayName {
			return nil, ErrInvalidCommunityDescriptionDuplicatedName
		}
	}

	// Sets the chat position to be the last within its category
	chat.Position = 0
	for _, c := range o.config.CommunityDescription.Chats {
		if c.CategoryId == chat.CategoryId {
			chat.Position++
		}
	}

	o.config.CommunityDescription.Chats[chatID] = chat

	o.increaseClock()

	changes := o.emptyCommunityChanges()
	changes.ChatsAdded[chatID] = chat
	return changes, nil
}

func (o *Community) EditChat(chatID string, chat *protobuf.CommunityChat) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.IsAdmin() {
		return nil, ErrNotAdmin
	}

	err := validateCommunityChat(o.config.CommunityDescription, chat)
	if err != nil {
		return nil, err
	}

	if o.config.CommunityDescription.Chats == nil {
		o.config.CommunityDescription.Chats = make(map[string]*protobuf.CommunityChat)
	}
	if _, exists := o.config.CommunityDescription.Chats[chatID]; !exists {
		return nil, ErrChatNotFound
	}

	o.config.CommunityDescription.Chats[chatID] = chat

	o.increaseClock()

	changes := o.emptyCommunityChanges()
	changes.ChatsModified[chatID] = &CommunityChatChanges{
		ChatModified: chat,
	}

	return changes, nil
}

func (o *Community) DeleteChat(chatID string) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}

	if o.config.CommunityDescription.Chats == nil {
		o.config.CommunityDescription.Chats = make(map[string]*protobuf.CommunityChat)
	}

	changes := o.emptyCommunityChanges()

	if chat, exists := o.config.CommunityDescription.Chats[chatID]; exists {
		tmpCatID := chat.CategoryId
		chat.CategoryId = ""
		o.SortCategoryChats(changes, tmpCatID)
	}

	delete(o.config.CommunityDescription.Chats, chatID)

	o.increaseClock()

	return o.config.CommunityDescription, nil
}

func (o *Community) InviteUserToOrg(pk *ecdsa.PublicKey) (*protobuf.CommunityInvitation, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}
	memberKey := common.PubkeyToHex(pk)

	if o.config.CommunityDescription.Members == nil {
		o.config.CommunityDescription.Members = make(map[string]*protobuf.CommunityMember)
	}

	if _, ok := o.config.CommunityDescription.Members[memberKey]; !ok {
		o.config.CommunityDescription.Members[memberKey] = &protobuf.CommunityMember{}
	}

	o.increaseClock()

	response := &protobuf.CommunityInvitation{}
	marshaledCommunity, err := o.toBytes()
	if err != nil {
		return nil, err
	}
	response.CommunityDescription = marshaledCommunity

	grant, err := o.buildGrant(pk, "")
	if err != nil {
		return nil, err
	}
	response.Grant = grant
	response.PublicKey = crypto.CompressPubkey(pk)

	return response, nil
}

func (o *Community) InviteUserToChat(pk *ecdsa.PublicKey, chatID string) (*protobuf.CommunityInvitation, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}
	memberKey := common.PubkeyToHex(pk)

	if _, ok := o.config.CommunityDescription.Members[memberKey]; !ok {
		o.config.CommunityDescription.Members[memberKey] = &protobuf.CommunityMember{}
	}

	chat, ok := o.config.CommunityDescription.Chats[chatID]
	if !ok {
		return nil, ErrChatNotFound
	}

	if chat.Members == nil {
		chat.Members = make(map[string]*protobuf.CommunityMember)
	}
	chat.Members[memberKey] = &protobuf.CommunityMember{}

	o.increaseClock()

	response := &protobuf.CommunityInvitation{}
	marshaledCommunity, err := o.toBytes()
	if err != nil {
		return nil, err
	}
	response.CommunityDescription = marshaledCommunity

	grant, err := o.buildGrant(pk, chatID)
	if err != nil {
		return nil, err
	}
	response.Grant = grant
	response.ChatId = chatID

	return response, nil
}

func (o *Community) getMember(pk *ecdsa.PublicKey) *protobuf.CommunityMember {

	key := common.PubkeyToHex(pk)
	member := o.config.CommunityDescription.Members[key]
	return member
}

func (o *Community) hasMember(pk *ecdsa.PublicKey) bool {

	member := o.getMember(pk)
	return member != nil
}

func (o *Community) IsBanned(pk *ecdsa.PublicKey) bool {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	return o.isBanned(pk)
}

func (o *Community) isBanned(pk *ecdsa.PublicKey) bool {
	key := common.PubkeyToHex(pk)

	for _, k := range o.config.CommunityDescription.BanList {
		if k == key {
			return true
		}
	}
	return false
}

func (o *Community) hasMemberPermission(member *protobuf.CommunityMember, permissions map[protobuf.CommunityMember_Roles]bool) bool {
	for _, r := range member.Roles {
		if permissions[r] {
			return true
		}
	}
	return false
}

func (o *Community) hasPermission(pk *ecdsa.PublicKey, roles map[protobuf.CommunityMember_Roles]bool) bool {
	member := o.getMember(pk)
	if member == nil {
		return false
	}

	return o.hasMemberPermission(member, roles)
}

func (o *Community) HasMember(pk *ecdsa.PublicKey) bool {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	return o.hasMember(pk)
}

func (o *Community) IsMemberInChat(pk *ecdsa.PublicKey, chatID string) bool {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.hasMember(pk) {
		return false
	}

	chat, ok := o.config.CommunityDescription.Chats[chatID]
	if !ok {
		return false
	}

	key := common.PubkeyToHex(pk)
	_, ok = chat.Members[key]
	return ok
}

func (o *Community) RemoveUserFromChat(pk *ecdsa.PublicKey, chatID string) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}
	if !o.hasMember(pk) {
		return o.config.CommunityDescription, nil
	}

	chat, ok := o.config.CommunityDescription.Chats[chatID]
	if !ok {
		return o.config.CommunityDescription, nil
	}

	key := common.PubkeyToHex(pk)
	delete(chat.Members, key)

	return o.config.CommunityDescription, nil
}

func (o *Community) RemoveUserFromOrg(pk *ecdsa.PublicKey) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}
	if !o.hasMember(pk) {
		return o.config.CommunityDescription, nil
	}
	key := common.PubkeyToHex(pk)

	// Remove from org
	delete(o.config.CommunityDescription.Members, key)

	// Remove from chats
	for _, chat := range o.config.CommunityDescription.Chats {
		delete(chat.Members, key)
	}

	o.increaseClock()

	return o.config.CommunityDescription, nil
}

func (o *Community) BanUserFromCommunity(pk *ecdsa.PublicKey) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}
	key := common.PubkeyToHex(pk)
	if o.hasMember(pk) {
		// Remove from org
		delete(o.config.CommunityDescription.Members, key)

		// Remove from chats
		for _, chat := range o.config.CommunityDescription.Chats {
			delete(chat.Members, key)
		}
	}

	found := false
	for _, u := range o.config.CommunityDescription.BanList {
		if u == key {
			found = true
		}
	}
	if !found {
		o.config.CommunityDescription.BanList = append(o.config.CommunityDescription.BanList, key)
	}

	o.increaseClock()

	return o.config.CommunityDescription, nil
}

func (o *Community) Edit(description *protobuf.CommunityDescription) {
	o.config.CommunityDescription.Identity.DisplayName = description.Identity.DisplayName
	o.config.CommunityDescription.Identity.Description = description.Identity.Description
	o.config.CommunityDescription.Identity.Color = description.Identity.Color
	o.config.CommunityDescription.Identity.Emoji = description.Identity.Emoji
	o.config.CommunityDescription.Identity.Images = description.Identity.Images
	o.increaseClock()
}

func (o *Community) Join() {
	o.config.Joined = true
}

func (o *Community) Leave() {
	o.config.Joined = false
}

func (o *Community) Joined() bool {
	return o.config.Joined
}

func (o *Community) Verified() bool {
	return o.config.Verified
}

func (o *Community) Muted() bool {
	return o.config.Muted
}

func (o *Community) MemberIdentity() *ecdsa.PublicKey {
	return o.config.MemberIdentity
}

// UpdateCommunityDescription will update the community to the new community description and return a list of changes
func (o *Community) UpdateCommunityDescription(signer *ecdsa.PublicKey, description *protobuf.CommunityDescription, rawMessage []byte) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !common.IsPubKeyEqual(o.config.ID, signer) {
		return nil, ErrNotAuthorized
	}

	err := ValidateCommunityDescription(description)
	if err != nil {
		return nil, err
	}

	response := o.emptyCommunityChanges()

	if description.Clock <= o.config.CommunityDescription.Clock {
		return response, nil
	}

	// We only calculate changes if we joined the community or we requested access, otherwise not interested
	if o.config.Joined || o.config.RequestedToJoinAt > 0 {
		// Check for new members at the org level
		for pk, member := range description.Members {
			if _, ok := o.config.CommunityDescription.Members[pk]; !ok {
				if response.MembersAdded == nil {
					response.MembersAdded = make(map[string]*protobuf.CommunityMember)
				}
				response.MembersAdded[pk] = member
			}
		}

		// Check for removed members at the org level
		for pk, member := range o.config.CommunityDescription.Members {
			if _, ok := description.Members[pk]; !ok {
				if response.MembersRemoved == nil {
					response.MembersRemoved = make(map[string]*protobuf.CommunityMember)
				}
				response.MembersRemoved[pk] = member
			}
		}

		// check for removed chats
		for chatID, chat := range o.config.CommunityDescription.Chats {
			if description.Chats == nil {
				description.Chats = make(map[string]*protobuf.CommunityChat)
			}
			if _, ok := description.Chats[chatID]; !ok {
				if response.ChatsRemoved == nil {
					response.ChatsRemoved = make(map[string]*protobuf.CommunityChat)
				}

				response.ChatsRemoved[chatID] = chat
			}
		}

		for chatID, chat := range description.Chats {
			if o.config.CommunityDescription.Chats == nil {
				o.config.CommunityDescription.Chats = make(map[string]*protobuf.CommunityChat)
			}

			if _, ok := o.config.CommunityDescription.Chats[chatID]; !ok {
				if response.ChatsAdded == nil {
					response.ChatsAdded = make(map[string]*protobuf.CommunityChat)
				}

				response.ChatsAdded[chatID] = chat
			} else {
				// Check for members added
				for pk, member := range description.Chats[chatID].Members {
					if _, ok := o.config.CommunityDescription.Chats[chatID].Members[pk]; !ok {
						if response.ChatsModified[chatID] == nil {
							response.ChatsModified[chatID] = &CommunityChatChanges{
								MembersAdded:   make(map[string]*protobuf.CommunityMember),
								MembersRemoved: make(map[string]*protobuf.CommunityMember),
							}
						}

						response.ChatsModified[chatID].MembersAdded[pk] = member
					}
				}

				// check for members removed
				for pk, member := range o.config.CommunityDescription.Chats[chatID].Members {
					if _, ok := description.Chats[chatID].Members[pk]; !ok {
						if response.ChatsModified[chatID] == nil {
							response.ChatsModified[chatID] = &CommunityChatChanges{
								MembersAdded:   make(map[string]*protobuf.CommunityMember),
								MembersRemoved: make(map[string]*protobuf.CommunityMember),
							}
						}

						response.ChatsModified[chatID].MembersRemoved[pk] = member
					}
				}
			}
		}

		// Check for categories that were removed
		for categoryID := range o.config.CommunityDescription.Categories {
			if description.Categories == nil {
				description.Categories = make(map[string]*protobuf.CommunityCategory)
			}

			if description.Chats == nil {
				description.Chats = make(map[string]*protobuf.CommunityChat)
			}

			if _, ok := description.Categories[categoryID]; !ok {
				response.CategoriesRemoved = append(response.CategoriesRemoved, categoryID)
			}

			if o.config.CommunityDescription.Chats == nil {
				o.config.CommunityDescription.Chats = make(map[string]*protobuf.CommunityChat)
			}
		}

		// Check for categories that were added
		for categoryID, category := range description.Categories {
			if o.config.CommunityDescription.Categories == nil {
				o.config.CommunityDescription.Categories = make(map[string]*protobuf.CommunityCategory)
			}
			if _, ok := o.config.CommunityDescription.Categories[categoryID]; !ok {
				if response.CategoriesAdded == nil {
					response.CategoriesAdded = make(map[string]*protobuf.CommunityCategory)
				}

				response.CategoriesAdded[categoryID] = category
			} else {
				if o.config.CommunityDescription.Categories[categoryID].Name != category.Name || o.config.CommunityDescription.Categories[categoryID].Position != category.Position {
					response.CategoriesModified[categoryID] = category
				}
			}
		}

		// Check for chat categories that were modified
		for chatID, chat := range description.Chats {
			if o.config.CommunityDescription.Chats == nil {
				o.config.CommunityDescription.Chats = make(map[string]*protobuf.CommunityChat)
			}

			if _, ok := o.config.CommunityDescription.Chats[chatID]; !ok {
				continue // It's a new chat
			}

			if o.config.CommunityDescription.Chats[chatID].CategoryId != chat.CategoryId {
				if response.ChatsModified[chatID] == nil {
					response.ChatsModified[chatID] = &CommunityChatChanges{
						MembersAdded:   make(map[string]*protobuf.CommunityMember),
						MembersRemoved: make(map[string]*protobuf.CommunityMember),
					}
				}

				response.ChatsModified[chatID].CategoryModified = chat.CategoryId
			}
		}
	}

	o.config.CommunityDescription = description
	o.config.MarshaledCommunityDescription = rawMessage

	return response, nil
}

// ValidateRequestToJoin validates a request, checks that the right permissions are applied
func (o *Community) ValidateRequestToJoin(signer *ecdsa.PublicKey, request *protobuf.CommunityRequestToJoin) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// If we are not admin, fuggetaboutit
	if !o.IsAdmin() {
		return ErrNotAdmin
	}

	// If the org is ens name only, then reject if not present
	if o.config.CommunityDescription.Permissions.EnsOnly && len(request.EnsName) == 0 {
		return ErrCantRequestAccess
	}

	if len(request.ChatId) != 0 {
		return o.validateRequestToJoinWithChatID(request)
	}

	err := o.validateRequestToJoinWithoutChatID(request)
	if err != nil {
		return err
	}

	return nil
}

func (o *Community) IsAdmin() bool {
	return o.config.PrivateKey != nil
}

func (o *Community) IsMemberAdmin(publicKey *ecdsa.PublicKey) bool {
	return o.hasPermission(publicKey, adminRolePermissions())
}

func canManageUsersRolePermissions() map[protobuf.CommunityMember_Roles]bool {
	roles := adminRolePermissions()
	roles[protobuf.CommunityMember_ROLE_MANAGE_USERS] = true
	return roles
}

func adminRolePermissions() map[protobuf.CommunityMember_Roles]bool {
	roles := make(map[protobuf.CommunityMember_Roles]bool)
	roles[protobuf.CommunityMember_ROLE_ALL] = true
	return roles
}

func (o *Community) validateRequestToJoinWithChatID(request *protobuf.CommunityRequestToJoin) error {

	chat, ok := o.config.CommunityDescription.Chats[request.ChatId]

	if !ok {
		return ErrChatNotFound
	}

	// If chat is no permissions, access should not have been requested
	if chat.Permissions.Access != protobuf.CommunityPermissions_ON_REQUEST {
		return ErrCantRequestAccess
	}

	if chat.Permissions.EnsOnly && len(request.EnsName) == 0 {
		return ErrCantRequestAccess
	}

	return nil
}

func (o *Community) OnRequest() bool {
	return o.config.CommunityDescription.Permissions.Access == protobuf.CommunityPermissions_ON_REQUEST
}

func (o *Community) InvitationOnly() bool {
	return o.config.CommunityDescription.Permissions.Access == protobuf.CommunityPermissions_INVITATION_ONLY
}

func (o *Community) validateRequestToJoinWithoutChatID(request *protobuf.CommunityRequestToJoin) error {

	// If they want access to the org only, check that the org is ON_REQUEST
	if o.config.CommunityDescription.Permissions.Access != protobuf.CommunityPermissions_ON_REQUEST {
		return ErrCantRequestAccess
	}

	return nil
}

func (o *Community) ID() types.HexBytes {
	return crypto.CompressPubkey(o.config.ID)
}

func (o *Community) IDString() string {
	return types.EncodeHex(o.ID())
}

func (o *Community) StatusUpdatesChannelID() string {
	return o.IDString() + "-ping"
}

func (o *Community) MagnetlinkMessageChannelID() string {
	return o.IDString() + "-magnetlinks"
}

func (o *Community) DefaultFilters() []string {
	cID := o.IDString()
	updatesChannelID := o.StatusUpdatesChannelID()
	mlChannelID := o.MagnetlinkMessageChannelID()
	return []string{cID, updatesChannelID, mlChannelID}
}

func (o *Community) PrivateKey() *ecdsa.PrivateKey {
	return o.config.PrivateKey
}

func (o *Community) PublicKey() *ecdsa.PublicKey {
	return o.config.ID
}

func (o *Community) Description() *protobuf.CommunityDescription {
	return o.config.CommunityDescription
}

func (o *Community) marshaledDescription() ([]byte, error) {
	return proto.Marshal(o.config.CommunityDescription)
}

func (o *Community) MarshaledDescription() ([]byte, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	return o.marshaledDescription()
}

func (o *Community) toBytes() ([]byte, error) {

	// This should not happen, as we can only serialize on our side if we
	// created the community
	if o.config.PrivateKey == nil && len(o.config.MarshaledCommunityDescription) == 0 {
		return nil, ErrNotAdmin
	}

	// We are not admin, use the received serialized version
	if o.config.PrivateKey == nil {
		return o.config.MarshaledCommunityDescription, nil
	}

	// serialize and sign
	payload, err := o.marshaledDescription()
	if err != nil {
		return nil, err
	}

	return protocol.WrapMessageV1(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, o.config.PrivateKey)
}

// ToBytes returns the community in a wrapped & signed protocol message
func (o *Community) ToBytes() ([]byte, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	return o.toBytes()
}

func (o *Community) Chats() map[string]*protobuf.CommunityChat {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	response := make(map[string]*protobuf.CommunityChat)
	for k, v := range o.config.CommunityDescription.Chats {
		response[k] = v
	}
	return response
}

func (o *Community) Categories() map[string]*protobuf.CommunityCategory {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	response := make(map[string]*protobuf.CommunityCategory)
	for k, v := range o.config.CommunityDescription.Categories {
		response[k] = v
	}
	return response
}

func (o *Community) VerifyGrantSignature(data []byte) (*protobuf.Grant, error) {
	if len(data) <= signatureLength {
		return nil, ErrInvalidGrant
	}
	signature := data[:signatureLength]
	payload := data[signatureLength:]
	grant := &protobuf.Grant{}
	err := proto.Unmarshal(payload, grant)
	if err != nil {
		return nil, err
	}

	if grant.Clock == 0 {
		return nil, ErrInvalidGrant
	}
	if grant.MemberId == nil {
		return nil, ErrInvalidGrant
	}
	if !bytes.Equal(grant.CommunityId, o.ID()) {
		return nil, ErrInvalidGrant
	}

	extractedPublicKey, err := crypto.SigToPub(crypto.Keccak256(payload), signature)
	if err != nil {
		return nil, err
	}

	if !common.IsPubKeyEqual(o.config.ID, extractedPublicKey) {
		return nil, ErrInvalidGrant
	}

	return grant, nil
}

func (o *Community) CanPost(pk *ecdsa.PublicKey, chatID string, grantBytes []byte) (bool, error) {
	if o.config.CommunityDescription.Chats == nil {
		o.config.Logger.Debug("canPost, no-chats")
		return false, nil
	}

	chat, ok := o.config.CommunityDescription.Chats[chatID]
	if !ok {
		o.config.Logger.Debug("canPost, no chat with id", zap.String("chat-id", chatID))
		return false, nil
	}

	// creator can always post
	if common.IsPubKeyEqual(pk, o.config.ID) {
		return true, nil
	}

	// if banned cannot post
	if o.isBanned(pk) {
		return false, nil
	}

	// If both the chat & the org have no permissions, the user is allowed to post
	if o.config.CommunityDescription.Permissions.Access == protobuf.CommunityPermissions_NO_MEMBERSHIP && chat.Permissions.Access == protobuf.CommunityPermissions_NO_MEMBERSHIP {
		return true, nil
	}

	if chat.Permissions.Access != protobuf.CommunityPermissions_NO_MEMBERSHIP {
		if chat.Members == nil {
			o.config.Logger.Debug("canPost, no members in chat", zap.String("chat-id", chatID))
			return false, nil
		}

		_, ok := chat.Members[common.PubkeyToHex(pk)]
		// If member, we stop here
		if ok {
			return true, nil
		}

		// If not a member, and not grant, we return
		if !ok && grantBytes == nil {
			o.config.Logger.Debug("canPost, not a member in chat", zap.String("chat-id", chatID))
			return false, nil
		}

		// Otherwise we verify the grant
		return o.canPostWithGrant(pk, chatID, grantBytes)
	}

	// Chat has no membership, check org permissions
	if o.config.CommunityDescription.Members == nil {
		o.config.Logger.Debug("canPost, no members in org", zap.String("chat-id", chatID))
		return false, nil
	}

	// If member, they can post
	_, ok = o.config.CommunityDescription.Members[common.PubkeyToHex(pk)]
	if ok {
		return true, nil
	}

	// Not a member and no grant, can't post
	if !ok && grantBytes == nil {
		o.config.Logger.Debug("canPost, not a member in org", zap.String("chat-id", chatID), zap.String("pubkey", common.PubkeyToHex(pk)))
		return false, nil
	}

	return o.canPostWithGrant(pk, chatID, grantBytes)
}

func (o *Community) canPostWithGrant(pk *ecdsa.PublicKey, chatID string, grantBytes []byte) (bool, error) {
	grant, err := o.VerifyGrantSignature(grantBytes)
	if err != nil {
		return false, err
	}
	// If the clock is lower or equal is invalid
	if grant.Clock <= o.config.CommunityDescription.Clock {
		return false, nil
	}

	if grant.MemberId == nil {
		return false, nil
	}

	grantPk, err := crypto.DecompressPubkey(grant.MemberId)
	if err != nil {
		return false, nil
	}

	if !common.IsPubKeyEqual(grantPk, pk) {
		return false, nil
	}

	if chatID != grant.ChatId {
		return false, nil
	}

	return true, nil
}

func (o *Community) buildGrant(key *ecdsa.PublicKey, chatID string) ([]byte, error) {
	grant := &protobuf.Grant{
		CommunityId: o.ID(),
		MemberId:    crypto.CompressPubkey(key),
		ChatId:      chatID,
		Clock:       o.config.CommunityDescription.Clock,
	}
	marshaledGrant, err := proto.Marshal(grant)
	if err != nil {
		return nil, err
	}

	signatureMaterial := crypto.Keccak256(marshaledGrant)

	signature, err := crypto.Sign(signatureMaterial, o.config.PrivateKey)
	if err != nil {
		return nil, err
	}

	return append(signature, marshaledGrant...), nil
}

func (o *Community) increaseClock() {
	o.config.CommunityDescription.Clock = o.nextClock()
}

func (o *Community) Clock() uint64 {
	return o.config.CommunityDescription.Clock
}

func (o *Community) CanRequestAccess(pk *ecdsa.PublicKey) bool {
	if o.hasMember(pk) {
		return false
	}

	if o.config.CommunityDescription == nil {
		return false
	}

	if o.config.CommunityDescription.Permissions == nil {
		return false
	}

	return o.config.CommunityDescription.Permissions.Access == protobuf.CommunityPermissions_ON_REQUEST
}

func (o *Community) CanManageUsers(pk *ecdsa.PublicKey) bool {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.IsAdmin() {
		return true
	}

	if !o.hasMember(pk) {
		return false
	}

	roles := canManageUsersRolePermissions()
	return o.hasPermission(pk, roles)

}
func (o *Community) isMember() bool {
	return o.hasMember(o.config.MemberIdentity)
}

func (o *Community) CanMemberIdentityPost(chatID string) (bool, error) {
	return o.CanPost(o.config.MemberIdentity, chatID, nil)
}

// CanJoin returns whether a user can join the community, only if it's
func (o *Community) canJoin() bool {
	if o.config.Joined {
		return false
	}

	if o.IsAdmin() {
		return true
	}

	if o.config.CommunityDescription.Permissions.Access == protobuf.CommunityPermissions_NO_MEMBERSHIP {
		return true
	}

	return o.isMember()
}

func (o *Community) RequestedToJoinAt() uint64 {
	return o.config.RequestedToJoinAt
}

func (o *Community) nextClock() uint64 {
	return o.config.CommunityDescription.Clock + 1
}

func (o *Community) CanManageUsersPublicKeys() ([]*ecdsa.PublicKey, error) {
	var response []*ecdsa.PublicKey
	roles := canManageUsersRolePermissions()
	for pkString, member := range o.config.CommunityDescription.Members {
		if o.hasMemberPermission(member, roles) {
			pk, err := common.HexToPubkey(pkString)
			if err != nil {
				return nil, err
			}

			response = append(response, pk)
		}

	}
	return response, nil
}

func (o *Community) AddRequestToJoin(request *RequestToJoin) {
	o.config.RequestsToJoin = append(o.config.RequestsToJoin, request)
}

func (o *Community) RequestsToJoin() []*RequestToJoin {
	return o.config.RequestsToJoin
}

func (o *Community) ChatIDs() (chatIDs []string) {
	for id := range o.config.CommunityDescription.Chats {
		chatIDs = append(chatIDs, o.IDString()+id)
	}
	return chatIDs
}

func emptyCommunityChanges() *CommunityChanges {
	return &CommunityChanges{
		MembersAdded:   make(map[string]*protobuf.CommunityMember),
		MembersRemoved: make(map[string]*protobuf.CommunityMember),

		ChatsRemoved:  make(map[string]*protobuf.CommunityChat),
		ChatsAdded:    make(map[string]*protobuf.CommunityChat),
		ChatsModified: make(map[string]*CommunityChatChanges),

		CategoriesRemoved:  []string{},
		CategoriesAdded:    make(map[string]*protobuf.CommunityCategory),
		CategoriesModified: make(map[string]*protobuf.CommunityCategory),
	}
}

type sortSlice []sorterHelperIdx
type sorterHelperIdx struct {
	pos    int32
	catID  string
	chatID string
}

func (d sortSlice) Len() int {
	return len(d)
}

func (d sortSlice) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d sortSlice) Less(i, j int) bool {
	return d[i].pos < d[j].pos
}
