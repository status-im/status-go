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
	"github.com/status-im/status-go/protocol/requests"
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
	Spectated                     bool
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

type CommunityAdminSettings struct {
	PinMessageAllMembersEnabled bool `json:"pinMessageAllMembersEnabled"`
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

type CommunityTag struct {
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
}

func (o *Community) MarshalPublicAPIJSON() ([]byte, error) {
	if o.config.MemberIdentity == nil {
		return nil, errors.New("member identity not set")
	}
	communityItem := struct {
		ID                      types.HexBytes                                `json:"id"`
		Verified                bool                                          `json:"verified"`
		Chats                   map[string]CommunityChat                      `json:"chats"`
		Categories              map[string]CommunityCategory                  `json:"categories"`
		Name                    string                                        `json:"name"`
		Description             string                                        `json:"description"`
		IntroMessage            string                                        `json:"introMessage"`
		OutroMessage            string                                        `json:"outroMessage"`
		Tags                    []CommunityTag                                `json:"tags"`
		Images                  map[string]images.IdentityImage               `json:"images"`
		Color                   string                                        `json:"color"`
		MembersCount            int                                           `json:"membersCount"`
		EnsName                 string                                        `json:"ensName"`
		Link                    string                                        `json:"link"`
		CommunityAdminSettings  CommunityAdminSettings                        `json:"adminSettings"`
		Encrypted               bool                                          `json:"encrypted"`
		BanList                 []string                                      `json:"banList"`
		TokenPermissions        map[string]*protobuf.CommunityTokenPermission `json:"tokenPermissions"`
		CommunityTokensMetadata []*protobuf.CommunityTokenMetadata            `json:"communityTokensMetadata"`
		ActiveMembersCount      uint64                                        `json:"activeMembersCount"`
	}{
		ID:         o.ID(),
		Verified:   o.config.Verified,
		Chats:      make(map[string]CommunityChat),
		Categories: make(map[string]CommunityCategory),
		Tags:       o.Tags(),
	}
	if o.config.CommunityDescription != nil {
		for id, c := range o.config.CommunityDescription.Categories {
			category := CommunityCategory{
				ID:       id,
				Name:     c.Name,
				Position: int(c.Position),
			}
			communityItem.Categories[id] = category
			communityItem.Encrypted = o.config.CommunityDescription.Encrypted
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

		communityItem.TokenPermissions = o.config.CommunityDescription.TokenPermissions
		communityItem.MembersCount = len(o.config.CommunityDescription.Members)
		communityItem.Link = fmt.Sprintf("https://join.status.im/c/0x%x", o.ID())
		communityItem.IntroMessage = o.config.CommunityDescription.IntroMessage
		communityItem.OutroMessage = o.config.CommunityDescription.OutroMessage
		communityItem.BanList = o.config.CommunityDescription.BanList
		communityItem.CommunityTokensMetadata = o.config.CommunityDescription.CommunityTokensMetadata
		communityItem.ActiveMembersCount = o.config.CommunityDescription.ActiveMembersCount

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

		communityItem.CommunityAdminSettings = CommunityAdminSettings{
			PinMessageAllMembersEnabled: false,
		}

		if o.config.CommunityDescription.AdminSettings != nil {
			communityItem.CommunityAdminSettings.PinMessageAllMembersEnabled = o.config.CommunityDescription.AdminSettings.PinMessageAllMembersEnabled
		}
	}
	return json.Marshal(communityItem)
}

func (o *Community) MarshalJSON() ([]byte, error) {
	if o.config.MemberIdentity == nil {
		return nil, errors.New("member identity not set")
	}
	communityItem := struct {
		ID                          types.HexBytes                                `json:"id"`
		Admin                       bool                                          `json:"admin"`
		Verified                    bool                                          `json:"verified"`
		Joined                      bool                                          `json:"joined"`
		Spectated                   bool                                          `json:"spectated"`
		RequestedAccessAt           int                                           `json:"requestedAccessAt"`
		Name                        string                                        `json:"name"`
		Description                 string                                        `json:"description"`
		IntroMessage                string                                        `json:"introMessage"`
		OutroMessage                string                                        `json:"outroMessage"`
		Tags                        []CommunityTag                                `json:"tags"`
		Chats                       map[string]CommunityChat                      `json:"chats"`
		Categories                  map[string]CommunityCategory                  `json:"categories"`
		Images                      map[string]images.IdentityImage               `json:"images"`
		Permissions                 *protobuf.CommunityPermissions                `json:"permissions"`
		Members                     map[string]*protobuf.CommunityMember          `json:"members"`
		CanRequestAccess            bool                                          `json:"canRequestAccess"`
		CanManageUsers              bool                                          `json:"canManageUsers"`
		CanDeleteMessageForEveryone bool                                          `json:"canDeleteMessageForEveryone"`
		CanJoin                     bool                                          `json:"canJoin"`
		Color                       string                                        `json:"color"`
		RequestedToJoinAt           uint64                                        `json:"requestedToJoinAt,omitempty"`
		IsMember                    bool                                          `json:"isMember"`
		Muted                       bool                                          `json:"muted"`
		CommunityAdminSettings      CommunityAdminSettings                        `json:"adminSettings"`
		Encrypted                   bool                                          `json:"encrypted"`
		BanList                     []string                                      `json:"banList"`
		TokenPermissions            map[string]*protobuf.CommunityTokenPermission `json:"tokenPermissions"`
		CommunityTokensMetadata     []*protobuf.CommunityTokenMetadata            `json:"communityTokensMetadata"`
		ActiveMembersCount          uint64                                        `json:"activeMembersCount"`
	}{
		ID:                          o.ID(),
		Admin:                       o.IsOwner(),
		Verified:                    o.config.Verified,
		Chats:                       make(map[string]CommunityChat),
		Categories:                  make(map[string]CommunityCategory),
		Joined:                      o.config.Joined,
		Spectated:                   o.config.Spectated,
		CanRequestAccess:            o.CanRequestAccess(o.config.MemberIdentity),
		CanJoin:                     o.canJoin(),
		CanManageUsers:              o.CanManageUsers(o.config.MemberIdentity),
		CanDeleteMessageForEveryone: o.CanDeleteMessageForEveryone(o.config.MemberIdentity),
		RequestedToJoinAt:           o.RequestedToJoinAt(),
		IsMember:                    o.isMember(),
		Muted:                       o.config.Muted,
		Tags:                        o.Tags(),
		Encrypted:                   o.Encrypted(),
	}
	if o.config.CommunityDescription != nil {
		for id, c := range o.config.CommunityDescription.Categories {
			category := CommunityCategory{
				ID:       id,
				Name:     c.Name,
				Position: int(c.Position),
			}
			communityItem.Encrypted = o.config.CommunityDescription.Encrypted
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
		communityItem.TokenPermissions = o.config.CommunityDescription.TokenPermissions
		communityItem.Members = o.config.CommunityDescription.Members
		communityItem.Permissions = o.config.CommunityDescription.Permissions
		communityItem.IntroMessage = o.config.CommunityDescription.IntroMessage
		communityItem.OutroMessage = o.config.CommunityDescription.OutroMessage
		communityItem.BanList = o.config.CommunityDescription.BanList
		communityItem.CommunityTokensMetadata = o.config.CommunityDescription.CommunityTokensMetadata
		communityItem.ActiveMembersCount = o.config.CommunityDescription.ActiveMembersCount

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

		communityItem.CommunityAdminSettings = CommunityAdminSettings{
			PinMessageAllMembersEnabled: false,
		}

		if o.config.CommunityDescription.AdminSettings != nil {
			communityItem.CommunityAdminSettings.PinMessageAllMembersEnabled = o.config.CommunityDescription.AdminSettings.PinMessageAllMembersEnabled
		}
	}
	return json.Marshal(communityItem)
}

func (o *Community) Identity() *protobuf.ChatIdentity {
	return o.config.CommunityDescription.Identity
}

func (o *Community) Permissions() *protobuf.CommunityPermissions {
	return o.config.CommunityDescription.Permissions
}

func (o *Community) AdminSettings() *protobuf.CommunityAdminSettings {
	return o.config.CommunityDescription.AdminSettings
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

func (o *Community) IntroMessage() string {
	if o != nil &&
		o.config != nil &&
		o.config.CommunityDescription != nil {
		return o.config.CommunityDescription.IntroMessage
	}
	return ""
}

func (o *Community) CommunityTokensMetadata() []*protobuf.CommunityTokenMetadata {
	if o != nil &&
		o.config != nil &&
		o.config.CommunityDescription != nil {
		return o.config.CommunityDescription.CommunityTokensMetadata
	}
	return nil
}

func (o *Community) Tags() []CommunityTag {
	if o != nil &&
		o.config != nil &&
		o.config.CommunityDescription != nil {
		var result []CommunityTag
		for _, t := range o.config.CommunityDescription.Tags {
			result = append(result, CommunityTag{
				Name:  t,
				Emoji: requests.TagsEmojies[t],
			})
		}
		return result
	}
	return nil
}

func (o *Community) TagsRaw() []string {
	return o.config.CommunityDescription.Tags
}

func (o *Community) OutroMessage() string {
	if o != nil &&
		o.config != nil &&
		o.config.CommunityDescription != nil {
		return o.config.CommunityDescription.OutroMessage
	}
	return ""
}

func (o *Community) Color() string {
	if o != nil &&
		o.config != nil &&
		o.config.CommunityDescription != nil &&
		o.config.CommunityDescription.Identity != nil {
		return o.config.CommunityDescription.Identity.Color
	}
	return ""
}

func (o *Community) Members() map[string]*protobuf.CommunityMember {
	if o != nil &&
		o.config != nil &&
		o.config.CommunityDescription != nil {
		return o.config.CommunityDescription.Members
	}
	return nil
}

func (o *Community) MembersCount() int {
	if o != nil &&
		o.config != nil &&
		o.config.CommunityDescription != nil {
		return len(o.config.CommunityDescription.Members)
	}
	return 0
}

func (o *Community) GetMemberPubkeys() []*ecdsa.PublicKey {
	if o != nil &&
		o.config != nil &&
		o.config.CommunityDescription != nil {
		pubkeys := make([]*ecdsa.PublicKey, len(o.config.CommunityDescription.Members))
		i := 0
		for hex := range o.config.CommunityDescription.Members {
			pubkeys[i], _ = common.HexToPubkey(hex)
			i++
		}
		return pubkeys
	}
	return nil
}
func (o *Community) initialize() {
	if o.config.CommunityDescription == nil {
		o.config.CommunityDescription = &protobuf.CommunityDescription{}

	}
}

type DeployState uint8

const (
	Failed DeployState = iota
	InProgress
	Deployed
)

type CommunityToken struct {
	TokenType          protobuf.CommunityTokenType `json:"tokenType"`
	CommunityID        string                      `json:"communityId"`
	Address            string                      `json:"address"`
	Name               string                      `json:"name"`
	Symbol             string                      `json:"symbol"`
	Description        string                      `json:"description"`
	Supply             int                         `json:"supply"`
	InfiniteSupply     bool                        `json:"infiniteSupply"`
	Transferable       bool                        `json:"transferable"`
	RemoteSelfDestruct bool                        `json:"remoteSelfDestruct"`
	ChainID            int                         `json:"chainId"`
	DeployState        DeployState                 `json:"deployState"`
	Base64Image        string                      `json:"image"`
}

type CommunitySettings struct {
	CommunityID                  string `json:"communityId"`
	HistoryArchiveSupportEnabled bool   `json:"historyArchiveSupportEnabled"`
	Clock                        uint64 `json:"clock"`
}

type CommunityChatChanges struct {
	ChatModified                  *protobuf.CommunityChat
	MembersAdded                  map[string]*protobuf.CommunityMember
	MembersRemoved                map[string]*protobuf.CommunityMember
	CategoryModified              string
	PositionModified              int
	FirstMessageTimestampModified uint32
}

type CommunityChanges struct {
	Community      *Community                           `json:"community"`
	MembersAdded   map[string]*protobuf.CommunityMember `json:"membersAdded"`
	MembersRemoved map[string]*protobuf.CommunityMember `json:"membersRemoved"`

	TokenPermissionsAdded    map[string]*protobuf.CommunityTokenPermission `json:"tokenPermissionsAdded"`
	TokenPermissionsModified map[string]*protobuf.CommunityTokenPermission `json:"tokenPermissionsModified"`
	TokenPermissionsRemoved  []string                                      `json:"tokenPermissionsRemoved"`

	ChatsRemoved  map[string]*protobuf.CommunityChat `json:"chatsRemoved"`
	ChatsAdded    map[string]*protobuf.CommunityChat `json:"chatsAdded"`
	ChatsModified map[string]*CommunityChatChanges   `json:"chatsModified"`

	CategoriesRemoved  []string                               `json:"categoriesRemoved"`
	CategoriesAdded    map[string]*protobuf.CommunityCategory `json:"categoriesAdded"`
	CategoriesModified map[string]*protobuf.CommunityCategory `json:"categoriesModified"`

	MemberWalletsRemoved []string            `json:"memberWalletsRemoved"`
	MemberWalletsAdded   map[string][]string `json:"memberWalletsAdded"`

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

	err := o.AddMember(pk, []protobuf.CommunityMember_Roles{})
	if err != nil {
		return nil, err
	}

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
	if common.IsPubKeyEqual(pk, o.config.ID) {
		return true
	}

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

func (o *Community) removeMemberFromOrg(pk *ecdsa.PublicKey) {
	if !o.hasMember(pk) {
		return
	}

	key := common.PubkeyToHex(pk)

	// Remove from org
	delete(o.config.CommunityDescription.Members, key)

	// Remove from chats
	for _, chat := range o.config.CommunityDescription.Chats {
		delete(chat.Members, key)
	}

	o.increaseClock()
}

func (o *Community) RemoveOurselvesFromOrg(pk *ecdsa.PublicKey) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.removeMemberFromOrg(pk)
}

func (o *Community) RemoveUserFromOrg(pk *ecdsa.PublicKey) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}

	o.removeMemberFromOrg(pk)
	return o.config.CommunityDescription, nil
}

func (o *Community) AddCommunityTokensMetadata(token *protobuf.CommunityTokenMetadata) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}
	o.config.CommunityDescription.CommunityTokensMetadata = append(o.config.CommunityDescription.CommunityTokensMetadata, token)
	o.increaseClock()

	return o.config.CommunityDescription, nil
}

func (o *Community) UnbanUserFromCommunity(pk *ecdsa.PublicKey) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}
	key := common.PubkeyToHex(pk)

	for i, v := range o.config.CommunityDescription.BanList {
		if v == key {
			o.config.CommunityDescription.BanList =
				append(o.config.CommunityDescription.BanList[:i], o.config.CommunityDescription.BanList[i+1:]...)
			break
		}
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

func (o *Community) AddRoleToMember(pk *ecdsa.PublicKey, role protobuf.CommunityMember_Roles) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}

	updated := false
	member := o.getMember(pk)
	if member != nil {
		roles := make(map[protobuf.CommunityMember_Roles]bool)
		roles[role] = true
		if !o.hasMemberPermission(member, roles) {
			member.Roles = append(member.Roles, role)
			o.config.CommunityDescription.Members[common.PubkeyToHex(pk)] = member
			updated = true
		}
	}

	if updated {
		o.increaseClock()
	}
	return o.config.CommunityDescription, nil
}

func (o *Community) RemoveRoleFromMember(pk *ecdsa.PublicKey, role protobuf.CommunityMember_Roles) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}

	updated := false
	member := o.getMember(pk)
	if member != nil {
		roles := make(map[protobuf.CommunityMember_Roles]bool)
		roles[role] = true
		if o.hasMemberPermission(member, roles) {
			var newRoles []protobuf.CommunityMember_Roles
			for _, r := range member.Roles {
				if r != role {
					newRoles = append(newRoles, r)
				}
			}
			member.Roles = newRoles
			o.config.CommunityDescription.Members[common.PubkeyToHex(pk)] = member
			updated = true
		}
	}

	if updated {
		o.increaseClock()
	}
	return o.config.CommunityDescription, nil
}

func (o *Community) Edit(description *protobuf.CommunityDescription) {
	o.config.CommunityDescription.Identity.DisplayName = description.Identity.DisplayName
	o.config.CommunityDescription.Identity.Description = description.Identity.Description
	o.config.CommunityDescription.Identity.Color = description.Identity.Color
	o.config.CommunityDescription.Tags = description.Tags
	o.config.CommunityDescription.Identity.Emoji = description.Identity.Emoji
	o.config.CommunityDescription.Identity.Images = description.Identity.Images
	o.config.CommunityDescription.IntroMessage = description.IntroMessage
	o.config.CommunityDescription.OutroMessage = description.OutroMessage
	if o.config.CommunityDescription.AdminSettings == nil {
		o.config.CommunityDescription.AdminSettings = &protobuf.CommunityAdminSettings{}
	}
	o.config.CommunityDescription.Permissions = description.Permissions
	o.config.CommunityDescription.AdminSettings.PinMessageAllMembersEnabled = description.AdminSettings.PinMessageAllMembersEnabled
	o.increaseClock()
}

func (o *Community) Join() {
	o.config.Joined = true
}

func (o *Community) Leave() {
	o.config.Joined = false
	o.config.Spectated = false
}

func (o *Community) Spectate() {
	o.config.Spectated = true
}

func (o *Community) Encrypted() bool {
	return o.config.CommunityDescription.Encrypted
}

func (o *Community) SetEncrypted(encrypted bool) {
	o.config.CommunityDescription.Encrypted = encrypted
}

func (o *Community) Joined() bool {
	return o.config.Joined
}

func (o *Community) Spectated() bool {
	return o.config.Spectated
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

	// This is done in case tags are updated and a client sends unknown tags
	description.Tags = requests.RemoveUnknownAndDeduplicateTags(description.Tags)

	err := ValidateCommunityDescription(description)
	if err != nil {
		return nil, err
	}

	response := o.emptyCommunityChanges()

	if description.Clock <= o.config.CommunityDescription.Clock {
		return response, nil
	}

	// We only calculate changes if we joined/spectated the community or we requested access, otherwise not interested
	if o.config.Joined || o.config.Spectated || o.config.RequestedToJoinAt > 0 {
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

				// check if first message timestamp was modified
				if o.config.CommunityDescription.Chats[chatID].Identity.FirstMessageTimestamp !=
					description.Chats[chatID].Identity.FirstMessageTimestamp {
					if response.ChatsModified[chatID] == nil {
						response.ChatsModified[chatID] = &CommunityChatChanges{
							MembersAdded:   make(map[string]*protobuf.CommunityMember),
							MembersRemoved: make(map[string]*protobuf.CommunityMember),
						}
					}
					response.ChatsModified[chatID].FirstMessageTimestampModified = description.Chats[chatID].Identity.FirstMessageTimestamp
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

func (o *Community) UpdateChatFirstMessageTimestamp(chatID string, timestamp uint32) (*CommunityChanges, error) {
	if !o.IsAdmin() {
		return nil, ErrNotAdmin
	}

	chat, ok := o.config.CommunityDescription.Chats[chatID]
	if !ok {
		return nil, ErrChatNotFound
	}

	chat.Identity.FirstMessageTimestamp = timestamp

	communityChanges := o.emptyCommunityChanges()
	communityChanges.ChatsModified[chatID] = &CommunityChatChanges{
		FirstMessageTimestampModified: timestamp,
	}
	return communityChanges, nil
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

func (o *Community) IsOwner() bool {
	return o.config.PrivateKey != nil
}

func (o *Community) IsAdmin() bool {
	if o.config.PrivateKey != nil {
		return true
	}
	return o.IsMemberAdmin(o.config.MemberIdentity)
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
	roles[protobuf.CommunityMember_ROLE_ADMIN] = true
	return roles
}

func canDeleteMessageForEveryonePermissions() map[protobuf.CommunityMember_Roles]bool {
	roles := adminRolePermissions()
	roles[protobuf.CommunityMember_ROLE_MODERATE_CONTENT] = true
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

func (o *Community) AcceptRequestToJoinAutomatically() bool {
	// We no longer have the notion of "no membership", but for historical reasons
	// we use `NO_MEMBERSHIP` to determine wether requests to join should be automatically
	// accepted or not.
	return o.config.CommunityDescription.Permissions.Access == protobuf.CommunityPermissions_NO_MEMBERSHIP
}

func (o *Community) validateRequestToJoinWithoutChatID(request *protobuf.CommunityRequestToJoin) error {
	// Previously, requests to join a community where only necessary when the community
	// permissions were indeed set to `ON_REQUEST`.
	// Now, users always have to request access but can get accepted automatically
	// (if permissions are set to NO_MEMBERSHIP).
	//
	// Hence, not only do we check whether the community permissions are ON_REQUEST but
	// also NO_MEMBERSHIP.
	if o.config.CommunityDescription.Permissions.Access != protobuf.CommunityPermissions_ON_REQUEST && o.config.CommunityDescription.Permissions.Access != protobuf.CommunityPermissions_NO_MEMBERSHIP {
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

func (o *Community) MemberUpdateChannelID() string {
	return o.IDString() + "-memberUpdate"
}

func (o *Community) DefaultFilters() []string {
	cID := o.IDString()
	uncompressedPubKey := common.PubkeyToHex(o.config.ID)[2:]
	updatesChannelID := o.StatusUpdatesChannelID()
	mlChannelID := o.MagnetlinkMessageChannelID()
	memberUpdateChannelID := o.MemberUpdateChannelID()
	return []string{cID, uncompressedPubKey, updatesChannelID, mlChannelID, memberUpdateChannelID}
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
	response := make(map[string]*protobuf.CommunityChat)

	// Why are we checking here for nil, it should be the responsibility of the caller
	if o == nil {
		return response
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config != nil && o.config.CommunityDescription != nil {
		for k, v := range o.config.CommunityDescription.Chats {
			response[k] = v
		}
	}

	return response
}

func (o *Community) Images() map[string]*protobuf.IdentityImage {
	response := make(map[string]*protobuf.IdentityImage)

	// Why are we checking here for nil, it should be the responsibility of the caller
	if o == nil {
		return response
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config != nil && o.config.CommunityDescription != nil && o.config.CommunityDescription.Identity != nil {
		for k, v := range o.config.CommunityDescription.Identity.Images {
			response[k] = v
		}
	}

	return response
}

func (o *Community) Categories() map[string]*protobuf.CommunityCategory {
	response := make(map[string]*protobuf.CommunityCategory)

	if o == nil {
		return response
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config != nil && o.config.CommunityDescription != nil {
		for k, v := range o.config.CommunityDescription.Categories {
			response[k] = v
		}
	}

	return response
}

func (o *Community) TokenPermissions() map[string]*protobuf.CommunityTokenPermission {
	return o.config.CommunityDescription.TokenPermissions
}

func (o *Community) HasTokenPermissions() bool {
	return len(o.config.CommunityDescription.TokenPermissions) > 0
}

func (o *Community) TokenPermissionsByType(permissionType protobuf.CommunityTokenPermission_Type) []*protobuf.CommunityTokenPermission {
	permissions := make([]*protobuf.CommunityTokenPermission, 0)
	for _, tokenPermission := range o.TokenPermissions() {
		if tokenPermission.Type == permissionType {
			permissions = append(permissions, tokenPermission)
		}
	}
	return permissions
}

func (o *Community) canManageTokenPermission(permission *protobuf.CommunityTokenPermission) bool {
	return o.config.PrivateKey != nil || (o.IsAdmin() && permission.Type != protobuf.CommunityTokenPermission_BECOME_ADMIN)
}

func (o *Community) AddTokenPermission(permission *protobuf.CommunityTokenPermission) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.canManageTokenPermission(permission) {
		return nil, ErrNotEnoughPermissions
	}

	if o.config.CommunityDescription.TokenPermissions == nil {
		o.config.CommunityDescription.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission)
	}

	if _, exists := o.config.CommunityDescription.TokenPermissions[permission.Id]; exists {
		return nil, ErrTokenPermissionAlreadyExists
	}

	o.config.CommunityDescription.TokenPermissions[permission.Id] = permission

	o.increaseClock()
	changes := o.emptyCommunityChanges()

	if changes.TokenPermissionsAdded == nil {
		changes.TokenPermissionsAdded = make(map[string]*protobuf.CommunityTokenPermission)
	}
	changes.TokenPermissionsAdded[permission.Id] = permission

	return changes, nil
}

func (o *Community) UpdateTokenPermission(permissionID string, tokenPermission *protobuf.CommunityTokenPermission) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.canManageTokenPermission(tokenPermission) {
		return nil, ErrNotEnoughPermissions
	}

	if o.config.CommunityDescription.TokenPermissions == nil {
		o.config.CommunityDescription.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission)
	}
	if _, ok := o.config.CommunityDescription.TokenPermissions[permissionID]; !ok {
		return nil, ErrTokenPermissionNotFound
	}

	changes := o.emptyCommunityChanges()
	o.config.CommunityDescription.TokenPermissions[permissionID] = tokenPermission
	o.increaseClock()

	if changes.TokenPermissionsModified == nil {
		changes.TokenPermissionsModified = make(map[string]*protobuf.CommunityTokenPermission)
	}
	changes.TokenPermissionsModified[permissionID] = o.config.CommunityDescription.TokenPermissions[tokenPermission.Id]

	return changes, nil
}

func (o *Community) DeleteTokenPermission(permissionID string) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	tokenPermission, exists := o.config.CommunityDescription.TokenPermissions[permissionID]

	if !exists {
		return nil, ErrTokenPermissionNotFound
	}

	if !o.canManageTokenPermission(tokenPermission) {
		return nil, ErrNotEnoughPermissions
	}

	delete(o.config.CommunityDescription.TokenPermissions, permissionID)
	changes := o.emptyCommunityChanges()
	changes.TokenPermissionsRemoved = append(changes.TokenPermissionsRemoved, permissionID)
	o.increaseClock()
	return changes, nil
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

func (o *Community) BuildGrant(key *ecdsa.PublicKey, chatID string) ([]byte, error) {
	return o.buildGrant(key, chatID)
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

func (o *Community) CanDeleteMessageForEveryone(pk *ecdsa.PublicKey) bool {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.IsAdmin() {
		return true
	}

	if !o.hasMember(pk) {
		return false
	}
	roles := canDeleteMessageForEveryonePermissions()
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

func (o *Community) AddMember(publicKey *ecdsa.PublicKey, roles []protobuf.CommunityMember_Roles) error {
	if o.config.PrivateKey == nil {
		return ErrNotAdmin
	}

	memberKey := common.PubkeyToHex(publicKey)

	if o.config.CommunityDescription.Members == nil {
		o.config.CommunityDescription.Members = make(map[string]*protobuf.CommunityMember)
	}

	if _, ok := o.config.CommunityDescription.Members[memberKey]; !ok {
		o.config.CommunityDescription.Members[memberKey] = &protobuf.CommunityMember{Roles: roles}
	}
	o.increaseClock()
	return nil
}

func (o *Community) ChatIDs() (chatIDs []string) {
	for id := range o.config.CommunityDescription.Chats {
		chatIDs = append(chatIDs, o.IDString()+id)
	}
	return chatIDs
}

func (o *Community) AllowsAllMembersToPinMessage() bool {
	return o.config.CommunityDescription.AdminSettings != nil && o.config.CommunityDescription.AdminSettings.PinMessageAllMembersEnabled
}

func (o *Community) AddMemberWallet(memberID string, addresses []string) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}

	if _, ok := o.config.CommunityDescription.Members[memberID]; !ok {
		return nil, ErrMemberNotFound
	}

	o.config.CommunityDescription.Members[memberID].WalletAccounts = addresses
	o.increaseClock()

	changes := o.emptyCommunityChanges()
	changes.MemberWalletsAdded[memberID] = o.config.CommunityDescription.Members[memberID].WalletAccounts
	return changes, nil
}

func (o *Community) SetActiveMembersCount(activeMembersCount uint64) (updated bool, err error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return false, ErrNotAdmin
	}

	if activeMembersCount == o.config.CommunityDescription.ActiveMembersCount {
		return false, nil
	}

	o.config.CommunityDescription.ActiveMembersCount = activeMembersCount
	o.increaseClock()

	return true, nil
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

		MemberWalletsRemoved: []string{},
		MemberWalletsAdded:   make(map[string][]string),
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
