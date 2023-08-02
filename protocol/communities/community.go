package communities

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/services/wallet/bigint"
)

const signatureLength = 65

type Config struct {
	PrivateKey                          *ecdsa.PrivateKey
	CommunityDescription                *protobuf.CommunityDescription
	CommunityDescriptionProtocolMessage []byte // community in a wrapped & signed (by owner) protocol message
	ID                                  *ecdsa.PublicKey
	Joined                              bool
	Requested                           bool
	Verified                            bool
	Spectated                           bool
	Muted                               bool
	MuteTill                            time.Time
	Logger                              *zap.Logger
	RequestedToJoinAt                   uint64
	RequestsToJoin                      []*RequestToJoin
	MemberIdentity                      *ecdsa.PublicKey
	SyncedAt                            uint64
	EventsData                          *EventsData
}

type EventsData struct {
	EventsBaseCommunityDescription []byte
	Events                         []CommunityEvent
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
		MemberRole                  protobuf.CommunityMember_Roles                `json:"memberRole"`
		IsControlNode               bool                                          `json:"isControlNode"`
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
		CanManageUsers              bool                                          `json:"canManageUsers"`              //TODO: we can remove this
		CanDeleteMessageForEveryone bool                                          `json:"canDeleteMessageForEveryone"` //TODO: we can remove this
		CanJoin                     bool                                          `json:"canJoin"`
		Color                       string                                        `json:"color"`
		RequestedToJoinAt           uint64                                        `json:"requestedToJoinAt,omitempty"`
		IsMember                    bool                                          `json:"isMember"`
		Muted                       bool                                          `json:"muted"`
		MuteTill                    time.Time                                     `json:"muteTill,omitempty"`
		CommunityAdminSettings      CommunityAdminSettings                        `json:"adminSettings"`
		Encrypted                   bool                                          `json:"encrypted"`
		BanList                     []string                                      `json:"banList"`
		TokenPermissions            map[string]*protobuf.CommunityTokenPermission `json:"tokenPermissions"`
		CommunityTokensMetadata     []*protobuf.CommunityTokenMetadata            `json:"communityTokensMetadata"`
		ActiveMembersCount          uint64                                        `json:"activeMembersCount"`
	}{
		ID:                          o.ID(),
		MemberRole:                  o.MemberRole(o.MemberIdentity()),
		IsControlNode:               o.IsControlNode(),
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
		MuteTill:                    o.config.MuteTill,
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

func (o *Community) TagsIndices() []uint32 {
	var indices []uint32
	for _, t := range o.config.CommunityDescription.Tags {
		i := uint32(0)
		for k := range requests.TagsEmojies {
			if k == t {
				indices = append(indices, i)
			}
			i++
		}
	}

	return indices
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
	Supply             *bigint.BigInt              `json:"supply"`
	InfiniteSupply     bool                        `json:"infiniteSupply"`
	Transferable       bool                        `json:"transferable"`
	RemoteSelfDestruct bool                        `json:"remoteSelfDestruct"`
	ChainID            int                         `json:"chainId"`
	DeployState        DeployState                 `json:"deployState"`
	Base64Image        string                      `json:"image"`
	Decimals           int                         `json:"decimals"`
}

type CommunitySettings struct {
	CommunityID                  string `json:"communityId"`
	HistoryArchiveSupportEnabled bool   `json:"historyArchiveSupportEnabled"`
	Clock                        uint64 `json:"clock"`
}

// `CommunityAdminEventChanges contain additional changes that don't live on
// a `Community` but still have to be propagated to other admin and control nodes
type CommunityEventChanges struct {
	*CommunityChanges
	// `RejectedRequestsToJoin` is a map of signer keys to requests to join
	RejectedRequestsToJoin map[string]*protobuf.CommunityRequestToJoin `json:"rejectedRequestsToJoin"`
	// `AcceptedRequestsToJoin` is a map of signer keys to requests to join
	AcceptedRequestsToJoin map[string]*protobuf.CommunityRequestToJoin `json:"acceptedRequestsToJoin"`
}

func (o *Community) emptyCommunityChanges() *CommunityChanges {
	changes := EmptyCommunityChanges()
	changes.Community = o
	return changes
}

func (o *Community) CreateChat(chatID string, chat *protobuf.CommunityChat) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	isControlNode := o.IsControlNode()
	allowedToSendEvents := o.HasPermissionToSendCommunityEvents()

	if !isControlNode && !allowedToSendEvents {
		return nil, ErrNotAdmin
	}

	if allowedToSendEvents {
		err := o.addNewCommunityEvent(o.ToCreateChannelCommunityEvent(chatID, chat))
		if err != nil {
			return nil, err
		}
	}

	err := o.createChat(chatID, chat)
	if err != nil {
		return nil, err
	}

	if isControlNode {
		o.increaseClock()
	}

	changes := o.emptyCommunityChanges()
	changes.ChatsAdded[chatID] = chat
	return changes, nil
}

func (o *Community) EditChat(chatID string, chat *protobuf.CommunityChat) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	isControlNode := o.IsControlNode()
	allowedToSendEvents := o.HasPermissionToSendCommunityEvents()

	if !isControlNode && !allowedToSendEvents {
		return nil, ErrNotAdmin
	}

	if allowedToSendEvents {
		err := o.addNewCommunityEvent(o.ToEditChannelCommunityEvent(chatID, chat))
		if err != nil {
			return nil, err
		}
	}

	err := o.editChat(chatID, chat)
	if err != nil {
		return nil, err
	}

	if isControlNode {
		o.increaseClock()
	}

	changes := o.emptyCommunityChanges()
	changes.ChatsModified[chatID] = &CommunityChatChanges{
		ChatModified: chat,
	}

	return changes, nil
}

func (o *Community) DeleteChat(chatID string) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	isControlNode := o.IsControlNode()
	allowedToSendEvents := o.HasPermissionToSendCommunityEvents()

	if !isControlNode && !allowedToSendEvents {
		return nil, ErrNotAdmin
	}

	if allowedToSendEvents {
		err := o.addNewCommunityEvent(o.ToDeleteChannelCommunityEvent(chatID))
		if err != nil {
			return nil, err
		}
	}

	changes := o.deleteChat(chatID)

	if isControlNode {
		o.increaseClock()
	}

	return changes, nil
}

func (o *Community) InviteUserToOrg(pk *ecdsa.PublicKey) (*protobuf.CommunityInvitation, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.IsControlNode() {
		return nil, ErrNotControlNode
	}

	_, err := o.AddMember(pk, []protobuf.CommunityMember_Roles{})
	if err != nil {
		return nil, err
	}

	response := &protobuf.CommunityInvitation{}
	wrappedCommunity, err := o.toProtocolMessageBytes()
	if err != nil {
		return nil, err
	}
	response.WrappedCommunityDescription = wrappedCommunity

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

	if !o.IsControlNode() {
		return nil, ErrNotControlNode
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
	wrappedCommunity, err := o.toProtocolMessageBytes()
	if err != nil {
		return nil, err
	}
	response.WrappedCommunityDescription = wrappedCommunity

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

func (o *Community) GetMember(pk *ecdsa.PublicKey) *protobuf.CommunityMember {
	return o.getMember(pk)
}

func (o *Community) getChatMember(pk *ecdsa.PublicKey, chatID string) *protobuf.CommunityMember {
	if !o.hasMember(pk) {
		return nil
	}

	chat, ok := o.config.CommunityDescription.Chats[chatID]
	if !ok {
		return nil
	}

	key := common.PubkeyToHex(pk)
	return chat.Members[key]
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

func (o *Community) memberHasRoles(member *protobuf.CommunityMember, roles map[protobuf.CommunityMember_Roles]bool) bool {
	for _, r := range member.Roles {
		if roles[r] {
			return true
		}
	}
	return false
}

func (o *Community) hasRoles(pk *ecdsa.PublicKey, roles map[protobuf.CommunityMember_Roles]bool) bool {
	if pk == nil || o.config == nil || o.config.ID == nil {
		return false
	}

	member := o.getMember(pk)
	if member == nil {
		return false
	}

	return o.memberHasRoles(member, roles)
}

func (o *Community) HasMember(pk *ecdsa.PublicKey) bool {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	return o.hasMember(pk)
}

func (o *Community) IsMemberInChat(pk *ecdsa.PublicKey, chatID string) bool {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	return o.getChatMember(pk, chatID) != nil
}

func (o *Community) RemoveUserFromChat(pk *ecdsa.PublicKey, chatID string) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.IsControlNode() {
		return nil, ErrNotControlNode
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

	if o.IsControlNode() {
		o.increaseClock()
	}

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
}

func (o *Community) RemoveOurselvesFromOrg(pk *ecdsa.PublicKey) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.removeMemberFromOrg(pk)
	o.increaseClock()
}

func (o *Community) RemoveUserFromOrg(pk *ecdsa.PublicKey) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	isControlNode := o.IsControlNode()
	allowedToSendEvents := o.HasPermissionToSendCommunityEvents()

	if !isControlNode && !allowedToSendEvents {
		return nil, ErrNotAdmin
	}

	if !isControlNode && o.IsPrivilegedMember(pk) {
		return nil, ErrCannotRemoveOwnerOrAdmin
	}

	if allowedToSendEvents {
		err := o.addNewCommunityEvent(o.ToKickCommunityMemberCommunityEvent(common.PubkeyToHex(pk)))
		if err != nil {
			return nil, err
		}
	}

	o.removeMemberFromOrg(pk)

	if isControlNode {
		o.increaseClock()
	}

	return o.config.CommunityDescription, nil
}

func (o *Community) AddCommunityTokensMetadata(token *protobuf.CommunityTokenMetadata) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	if !o.IsControlNode() {
		return nil, ErrNotControlNode
	}
	o.config.CommunityDescription.CommunityTokensMetadata = append(o.config.CommunityDescription.CommunityTokensMetadata, token)
	o.increaseClock()

	return o.config.CommunityDescription, nil
}

func (o *Community) UnbanUserFromCommunity(pk *ecdsa.PublicKey) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	isControlNode := o.IsControlNode()
	allowedToSendEvents := o.HasPermissionToSendCommunityEvents()

	if !isControlNode && !allowedToSendEvents {
		return nil, ErrNotAdmin
	}

	if allowedToSendEvents {
		err := o.addNewCommunityEvent(o.ToUnbanCommunityMemberCommunityEvent(common.PubkeyToHex(pk)))
		if err != nil {
			return nil, err
		}
	}

	o.unbanUserFromCommunity(pk)

	if isControlNode {
		o.increaseClock()
	}

	return o.config.CommunityDescription, nil
}

func (o *Community) BanUserFromCommunity(pk *ecdsa.PublicKey) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	isControlNode := o.IsControlNode()
	allowedToSendEvents := o.HasPermissionToSendCommunityEvents()

	if !isControlNode && !allowedToSendEvents {
		return nil, ErrNotAdmin
	}

	if !isControlNode && o.IsPrivilegedMember(pk) {
		return nil, ErrCannotBanOwnerOrAdmin
	}

	if allowedToSendEvents {
		err := o.addNewCommunityEvent(o.ToBanCommunityMemberCommunityEvent(common.PubkeyToHex(pk)))
		if err != nil {
			return nil, err
		}
	}

	o.banUserFromCommunity(pk)

	if isControlNode {
		o.increaseClock()
	}

	return o.config.CommunityDescription, nil
}

func (o *Community) AddRoleToMember(pk *ecdsa.PublicKey, role protobuf.CommunityMember_Roles) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.IsControlNode() {
		return nil, ErrNotControlNode
	}

	updated := false
	addRole := func(member *protobuf.CommunityMember) {
		roles := make(map[protobuf.CommunityMember_Roles]bool)
		roles[role] = true
		if !o.memberHasRoles(member, roles) {
			member.Roles = append(member.Roles, role)
			updated = true
		}
	}

	member := o.getMember(pk)
	if member != nil {
		addRole(member)
	}

	for channelID := range o.chats() {
		chatMember := o.getChatMember(pk, channelID)
		if chatMember != nil {
			addRole(member)
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

	if !o.IsControlNode() {
		return nil, ErrNotControlNode
	}

	updated := false
	removeRole := func(member *protobuf.CommunityMember) {
		roles := make(map[protobuf.CommunityMember_Roles]bool)
		roles[role] = true
		if o.memberHasRoles(member, roles) {
			var newRoles []protobuf.CommunityMember_Roles
			for _, r := range member.Roles {
				if r != role {
					newRoles = append(newRoles, r)
				}
			}
			member.Roles = newRoles
			updated = true
		}
	}

	member := o.getMember(pk)
	if member != nil {
		removeRole(member)
	}

	for channelID := range o.chats() {
		chatMember := o.getChatMember(pk, channelID)
		if chatMember != nil {
			removeRole(member)
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
}

func (o *Community) Join() {
	o.config.Joined = true
	o.config.Spectated = false
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

func (o *Community) MuteTill() time.Time {
	return o.config.MuteTill
}

func (o *Community) MemberIdentity() *ecdsa.PublicKey {
	return o.config.MemberIdentity
}

// UpdateCommunityDescription will update the community to the new community description and return a list of changes
func (o *Community) UpdateCommunityDescription(description *protobuf.CommunityDescription, rawMessage []byte, allowEqualClock bool) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// This is done in case tags are updated and a client sends unknown tags
	description.Tags = requests.RemoveUnknownAndDeduplicateTags(description.Tags)

	err := ValidateCommunityDescription(description)
	if err != nil {
		return nil, err
	}

	response := o.emptyCommunityChanges()

	// allowEqualClock == true only if this was a description from the handling request to join sent by an admin
	if allowEqualClock {
		if description.Clock < o.config.CommunityDescription.Clock {
			return response, nil
		}
	} else if description.Clock <= o.config.CommunityDescription.Clock {
		return response, nil
	}

	// We only calculate changes if we joined/spectated the community or we requested access, otherwise not interested
	if o.config.Joined || o.config.Spectated || o.config.RequestedToJoinAt > 0 {
		response = EvaluateCommunityChanges(o.config.CommunityDescription, description)
		response.Community = o
	}

	o.config.CommunityDescription = description
	o.config.CommunityDescriptionProtocolMessage = rawMessage

	return response, nil
}

func (o *Community) UpdateChatFirstMessageTimestamp(chatID string, timestamp uint32) (*CommunityChanges, error) {
	if !o.IsControlNode() {
		return nil, ErrNotControlNode
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
	if !o.IsControlNode() && !o.HasPermissionToSendCommunityEvents() {
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

// ValidateRequestToJoin validates a request, checks that the right permissions are applied
func (o *Community) ValidateEditSharedAddresses(signer *ecdsa.PublicKey, request *protobuf.CommunityEditRevealedAccounts) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// If we are not owner, fuggetaboutit
	if !o.IsControlNode() {
		return ErrNotOwner
	}

	if len(request.RevealedAccounts) == 0 {
		return errors.New("no addresses were shared")
	}

	if request.Clock < o.config.CommunityDescription.Members[common.PubkeyToHex(signer)].LastUpdateClock {
		return errors.New("edit request is older than the last one we have. Ignore")
	}

	return nil
}

// We treat control node as an owner with community key
func (o *Community) IsControlNode() bool {
	return o.config.PrivateKey != nil
}

func (o *Community) IsOwnerWithoutCommunityKey() bool {
	return o.config.PrivateKey == nil && o.IsMemberOwner(o.config.MemberIdentity)
}

func (o *Community) GetPrivilegedMembers() []*ecdsa.PublicKey {
	privilegedMembers := make([]*ecdsa.PublicKey, 0)
	members := o.GetMemberPubkeys()
	for _, member := range members {
		if o.IsPrivilegedMember(member) {
			privilegedMembers = append(privilegedMembers, member)
		}
	}
	return privilegedMembers
}

func (o *Community) HasPermissionToSendCommunityEvents() bool {
	return !o.IsControlNode() && o.hasRoles(o.config.MemberIdentity, manageCommunityRoles())
}

func (o *Community) IsMemberOwner(publicKey *ecdsa.PublicKey) bool {
	return o.hasRoles(publicKey, ownerRole())
}

func (o *Community) IsMemberTokenMaster(publicKey *ecdsa.PublicKey) bool {
	return o.hasRoles(publicKey, tokenMasterRole())
}

func (o *Community) IsMemberAdmin(publicKey *ecdsa.PublicKey) bool {
	return o.hasRoles(publicKey, adminRole())
}

func (o *Community) IsPrivilegedMember(publicKey *ecdsa.PublicKey) bool {
	return o.hasRoles(publicKey, manageCommunityRoles())
}

func manageCommunityRoles() map[protobuf.CommunityMember_Roles]bool {
	roles := make(map[protobuf.CommunityMember_Roles]bool)
	roles[protobuf.CommunityMember_ROLE_OWNER] = true
	roles[protobuf.CommunityMember_ROLE_ADMIN] = true
	roles[protobuf.CommunityMember_ROLE_TOKEN_MASTER] = true
	return roles
}

func manageUsersRole() map[protobuf.CommunityMember_Roles]bool {
	roles := manageCommunityRoles()
	roles[protobuf.CommunityMember_ROLE_MANAGE_USERS] = true
	return roles
}

func ownerRole() map[protobuf.CommunityMember_Roles]bool {
	roles := make(map[protobuf.CommunityMember_Roles]bool)
	roles[protobuf.CommunityMember_ROLE_OWNER] = true
	return roles
}

func adminRole() map[protobuf.CommunityMember_Roles]bool {
	roles := make(map[protobuf.CommunityMember_Roles]bool)
	roles[protobuf.CommunityMember_ROLE_ADMIN] = true
	return roles
}

func tokenMasterRole() map[protobuf.CommunityMember_Roles]bool {
	roles := make(map[protobuf.CommunityMember_Roles]bool)
	roles[protobuf.CommunityMember_ROLE_TOKEN_MASTER] = true
	return roles
}

func moderateContentRole() map[protobuf.CommunityMember_Roles]bool {
	roles := manageCommunityRoles()
	roles[protobuf.CommunityMember_ROLE_MODERATE_CONTENT] = true
	return roles
}

func (o *Community) MemberRole(pubKey *ecdsa.PublicKey) protobuf.CommunityMember_Roles {
	if o.IsMemberOwner(pubKey) {
		return protobuf.CommunityMember_ROLE_OWNER
	} else if o.IsMemberTokenMaster(pubKey) {
		return protobuf.CommunityMember_ROLE_TOKEN_MASTER
	} else if o.IsMemberAdmin(pubKey) {
		return protobuf.CommunityMember_ROLE_ADMIN
	} else if o.CanManageUsers(pubKey) {
		return protobuf.CommunityMember_ROLE_MANAGE_USERS
	} else if o.CanDeleteMessageForEveryone(pubKey) {
		return protobuf.CommunityMember_ROLE_MODERATE_CONTENT
	}

	return protobuf.CommunityMember_ROLE_NONE
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

func (o *Community) toProtocolMessageBytes() ([]byte, error) {
	// This should not happen, as we can only serialize on our side if we
	// created the community
	if !o.IsControlNode() && len(o.config.CommunityDescriptionProtocolMessage) == 0 {
		return nil, ErrNotControlNode
	}

	// If we are not a control node, use the received serialized version
	if !o.IsControlNode() {
		return o.config.CommunityDescriptionProtocolMessage, nil
	}

	// serialize and sign
	payload, err := o.marshaledDescription()
	if err != nil {
		return nil, err
	}

	return protocol.WrapMessageV1(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, o.config.PrivateKey)
}

// ToProtocolMessageBytes returns the community in a wrapped & signed protocol message
func (o *Community) ToProtocolMessageBytes() ([]byte, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	return o.toProtocolMessageBytes()
}

func (o *Community) Chats() map[string]*protobuf.CommunityChat {
	// Why are we checking here for nil, it should be the responsibility of the caller
	if o == nil {
		return make(map[string]*protobuf.CommunityChat)
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	return o.chats()
}

func (o *Community) chats() map[string]*protobuf.CommunityChat {
	response := make(map[string]*protobuf.CommunityChat)

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
	return o.config.CommunityDescription.TokenPermissions != nil && len(o.config.CommunityDescription.TokenPermissions) > 0
}

func (o *Community) ChannelHasTokenPermissions(chatID string) bool {
	if !o.HasTokenPermissions() {
		return false
	}

	for _, tokenPermission := range o.TokenPermissions() {
		if includes(tokenPermission.ChatIds, chatID) {
			return true
		}
	}

	return false
}

func TokenPermissionsByType(permissions map[string]*protobuf.CommunityTokenPermission, permissionType protobuf.CommunityTokenPermission_Type) []*protobuf.CommunityTokenPermission {
	result := make([]*protobuf.CommunityTokenPermission, 0)
	for _, tokenPermission := range permissions {
		if tokenPermission.Type == permissionType {
			result = append(result, tokenPermission)
		}
	}
	return result
}

func (o *Community) TokenPermissionsByType(permissionType protobuf.CommunityTokenPermission_Type) []*protobuf.CommunityTokenPermission {
	return TokenPermissionsByType(o.TokenPermissions(), permissionType)
}

func (o *Community) ChannelTokenPermissionsByType(channelID string, permissionType protobuf.CommunityTokenPermission_Type) []*protobuf.CommunityTokenPermission {
	permissions := make([]*protobuf.CommunityTokenPermission, 0)
	for _, tokenPermission := range o.TokenPermissions() {
		if tokenPermission.Type == permissionType && includes(tokenPermission.ChatIds, channelID) {
			permissions = append(permissions, tokenPermission)
		}
	}
	return permissions
}

func includes(channelIDs []string, channelID string) bool {
	for _, id := range channelIDs {
		if id == channelID {
			return true
		}
	}
	return false
}

func (o *Community) updateEncrypted() {
	o.config.CommunityDescription.Encrypted = len(o.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER)) > 0
}

func (o *Community) AddTokenPermission(permission *protobuf.CommunityTokenPermission) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	isControlNode := o.IsControlNode()
	allowedToSendEvents := o.HasPermissionToSendCommunityEvents()

	if !isControlNode && !allowedToSendEvents || (allowedToSendEvents && permission.Type == protobuf.CommunityTokenPermission_BECOME_ADMIN) {
		return nil, ErrNotEnoughPermissions
	}

	changes, err := o.addTokenPermission(permission)
	if err != nil {
		return nil, err
	}

	if allowedToSendEvents {
		err := o.addNewCommunityEvent(o.ToCommunityTokenPermissionChangeCommunityEvent(permission))
		if err != nil {
			return nil, err
		}
	}

	if isControlNode {
		o.updateEncrypted()
		o.increaseClock()
	}

	return changes, nil
}

func (o *Community) UpdateTokenPermission(permissionID string, tokenPermission *protobuf.CommunityTokenPermission) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	isControlNode := o.IsControlNode()
	allowedToSendEvents := o.HasPermissionToSendCommunityEvents()

	if !isControlNode && !allowedToSendEvents || (allowedToSendEvents && tokenPermission.Type == protobuf.CommunityTokenPermission_BECOME_ADMIN) {
		return nil, ErrNotEnoughPermissions
	}

	changes, err := o.updateTokenPermission(tokenPermission)
	if err != nil {
		return nil, err
	}

	if allowedToSendEvents {
		err := o.addNewCommunityEvent(o.ToCommunityTokenPermissionChangeCommunityEvent(tokenPermission))
		if err != nil {
			return nil, err
		}
	}

	if isControlNode {
		o.updateEncrypted()
		o.increaseClock()
	}

	return changes, nil
}

func (o *Community) DeleteTokenPermission(permissionID string) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	permission, exists := o.config.CommunityDescription.TokenPermissions[permissionID]

	if !exists {
		return nil, ErrTokenPermissionNotFound
	}

	isControlNode := o.IsControlNode()
	allowedToSendEvents := o.HasPermissionToSendCommunityEvents()

	if !isControlNode && !allowedToSendEvents || (allowedToSendEvents && permission.Type == protobuf.CommunityTokenPermission_BECOME_ADMIN) {
		return nil, ErrNotEnoughPermissions
	}

	changes, err := o.deleteTokenPermission(permissionID)
	if err != nil {
		return nil, err
	}

	if allowedToSendEvents {
		err := o.addNewCommunityEvent(o.ToCommunityTokenPermissionDeleteCommunityEvent(permission))
		if err != nil {
			return nil, err
		}
	}

	if isControlNode {
		o.updateEncrypted()
		o.increaseClock()
	}

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
	bytes := make([]byte, 0)
	if o.config.PrivateKey != nil {
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

		bytes = append(signature, marshaledGrant...)
	}
	return bytes, nil
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

	if o.IsControlNode() {
		return true
	}

	if !o.hasMember(pk) {
		return false
	}

	roles := manageUsersRole()
	return o.hasRoles(pk, roles)

}

func (o *Community) CanDeleteMessageForEveryone(pk *ecdsa.PublicKey) bool {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.IsControlNode() {
		return true
	}

	if !o.hasMember(pk) {
		return false
	}

	roles := moderateContentRole()
	return o.hasRoles(pk, roles)
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

	if o.IsControlNode() {
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
	roles := manageUsersRole()
	for pkString, member := range o.config.CommunityDescription.Members {
		if o.memberHasRoles(member, roles) {
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

func (o *Community) AddMember(publicKey *ecdsa.PublicKey, roles []protobuf.CommunityMember_Roles) (*CommunityChanges, error) {
	if !o.IsControlNode() && !o.HasPermissionToSendCommunityEvents() {
		return nil, ErrNotAdmin
	}

	memberKey := common.PubkeyToHex(publicKey)
	changes := o.emptyCommunityChanges()

	if o.config.CommunityDescription.Members == nil {
		o.config.CommunityDescription.Members = make(map[string]*protobuf.CommunityMember)
	}

	if _, ok := o.config.CommunityDescription.Members[memberKey]; !ok {
		o.config.CommunityDescription.Members[memberKey] = &protobuf.CommunityMember{Roles: roles}
		changes.MembersAdded[memberKey] = o.config.CommunityDescription.Members[memberKey]
	}

	o.increaseClock()
	return changes, nil
}

func (o *Community) AddMemberToChat(chatID string, publicKey *ecdsa.PublicKey, roles []protobuf.CommunityMember_Roles) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.IsControlNode() && !o.HasPermissionToSendCommunityEvents() {
		return nil, ErrNotAuthorized
	}

	memberKey := common.PubkeyToHex(publicKey)
	changes := o.emptyCommunityChanges()

	chat, ok := o.config.CommunityDescription.Chats[chatID]
	if !ok {
		return nil, ErrChatNotFound
	}

	if chat.Members == nil {
		chat.Members = make(map[string]*protobuf.CommunityMember)
	}
	chat.Members[memberKey] = &protobuf.CommunityMember{
		Roles: roles,
	}
	changes.ChatsModified[chatID] = &CommunityChatChanges{
		ChatModified: chat,
		MembersAdded: map[string]*protobuf.CommunityMember{
			memberKey: chat.Members[memberKey],
		},
	}

	if o.IsControlNode() {
		o.increaseClock()
	}

	return changes, nil
}

func (o *Community) PopulateChatWithAllMembers(chatID string) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.IsControlNode() {
		return o.emptyCommunityChanges(), ErrNotControlNode
	}

	return o.populateChatWithAllMembers(chatID)
}

func (o *Community) populateChatWithAllMembers(chatID string) (*CommunityChanges, error) {
	result := o.emptyCommunityChanges()

	chat, exists := o.chats()[chatID]
	if !exists {
		return result, ErrChatNotFound
	}

	membersAdded := make(map[string]*protobuf.CommunityMember)
	for pubKey, member := range o.Members() {
		if chat.Members[pubKey] == nil {
			membersAdded[pubKey] = member
		}
	}
	result.ChatsModified[chatID] = &CommunityChatChanges{
		MembersAdded: membersAdded,
	}

	chat.Members = o.Members()
	o.increaseClock()

	return result, nil
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

func (o *Community) AddMemberRevealedAccounts(memberID string, accounts []*protobuf.RevealedAccount, clock uint64) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.IsControlNode() && !o.HasPermissionToSendCommunityEvents() {
		return nil, ErrNotAdmin
	}

	if _, ok := o.config.CommunityDescription.Members[memberID]; !ok {
		return nil, ErrMemberNotFound
	}

	o.config.CommunityDescription.Members[memberID].RevealedAccounts = accounts
	o.config.CommunityDescription.Members[memberID].LastUpdateClock = clock
	o.increaseClock()

	changes := o.emptyCommunityChanges()
	changes.MemberWalletsAdded[memberID] = o.config.CommunityDescription.Members[memberID].RevealedAccounts
	return changes, nil
}

func (o *Community) AddMemberWithRevealedAccounts(dbRequest *RequestToJoin, roles []protobuf.CommunityMember_Roles, accounts []*protobuf.RevealedAccount) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	isControlNode := o.IsControlNode()
	allowedToSendEvents := o.HasPermissionToSendCommunityEvents()

	if !isControlNode && !allowedToSendEvents {
		return nil, ErrNotAdmin
	}

	changes := o.addMemberWithRevealedAccounts(dbRequest.PublicKey, roles, accounts, dbRequest.Clock)

	if allowedToSendEvents {
		acceptedRequestsToJoin := make(map[string]*protobuf.CommunityRequestToJoin)
		acceptedRequestsToJoin[dbRequest.PublicKey] = dbRequest.ToCommunityRequestToJoinProtobuf()

		adminChanges := &CommunityEventChanges{
			CommunityChanges:       changes,
			AcceptedRequestsToJoin: acceptedRequestsToJoin,
		}
		err := o.addNewCommunityEvent(o.ToCommunityRequestToJoinAcceptCommunityEvent(adminChanges))
		if err != nil {
			return nil, err
		}
	}

	if isControlNode {
		o.increaseClock()
	}

	return changes, nil
}

func (o *Community) CreateDeepCopy() *Community {
	return &Community{
		config: &Config{
			PrivateKey:                          o.config.PrivateKey,
			CommunityDescription:                proto.Clone(o.config.CommunityDescription).(*protobuf.CommunityDescription),
			CommunityDescriptionProtocolMessage: o.config.CommunityDescriptionProtocolMessage,
			ID:                                  o.config.ID,
			Joined:                              o.config.Joined,
			Requested:                           o.config.Requested,
			Verified:                            o.config.Verified,
			Spectated:                           o.config.Spectated,
			Muted:                               o.config.Muted,
			Logger:                              o.config.Logger,
			RequestedToJoinAt:                   o.config.RequestedToJoinAt,
			RequestsToJoin:                      o.config.RequestsToJoin,
			MemberIdentity:                      o.config.MemberIdentity,
			SyncedAt:                            o.config.SyncedAt,
			EventsData:                          o.config.EventsData,
		},
	}
}

func (o *Community) SetActiveMembersCount(activeMembersCount uint64) (updated bool, err error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.IsControlNode() {
		return false, ErrNotControlNode
	}

	if activeMembersCount == o.config.CommunityDescription.ActiveMembersCount {
		return false, nil
	}

	o.config.CommunityDescription.ActiveMembersCount = activeMembersCount
	o.increaseClock()

	return true, nil
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

func (o *Community) unbanUserFromCommunity(pk *ecdsa.PublicKey) {
	key := common.PubkeyToHex(pk)
	for i, v := range o.config.CommunityDescription.BanList {
		if v == key {
			o.config.CommunityDescription.BanList =
				append(o.config.CommunityDescription.BanList[:i], o.config.CommunityDescription.BanList[i+1:]...)
			break
		}
	}
}

func (o *Community) banUserFromCommunity(pk *ecdsa.PublicKey) {
	key := common.PubkeyToHex(pk)
	if o.hasMember(pk) {
		// Remove from org
		delete(o.config.CommunityDescription.Members, key)

		// Remove from chats
		for _, chat := range o.config.CommunityDescription.Chats {
			delete(chat.Members, key)
		}
	}

	for _, u := range o.config.CommunityDescription.BanList {
		if u == key {
			return
		}
	}

	o.config.CommunityDescription.BanList = append(o.config.CommunityDescription.BanList, key)
}

func (o *Community) editChat(chatID string, chat *protobuf.CommunityChat) error {
	err := validateCommunityChat(o.config.CommunityDescription, chat)
	if err != nil {
		return err
	}

	if o.config.CommunityDescription.Chats == nil {
		o.config.CommunityDescription.Chats = make(map[string]*protobuf.CommunityChat)
	}
	if _, exists := o.config.CommunityDescription.Chats[chatID]; !exists {
		return ErrChatNotFound
	}

	o.config.CommunityDescription.Chats[chatID] = chat

	return nil
}

func (o *Community) createChat(chatID string, chat *protobuf.CommunityChat) error {
	err := validateCommunityChat(o.config.CommunityDescription, chat)
	if err != nil {
		return err
	}

	if o.config.CommunityDescription.Chats == nil {
		o.config.CommunityDescription.Chats = make(map[string]*protobuf.CommunityChat)
	}
	if _, ok := o.config.CommunityDescription.Chats[chatID]; ok {
		return ErrChatAlreadyExists
	}

	for _, c := range o.config.CommunityDescription.Chats {
		if chat.Identity.DisplayName == c.Identity.DisplayName {
			return ErrInvalidCommunityDescriptionDuplicatedName
		}
	}

	// Sets the chat position to be the last within its category
	chat.Position = 0
	for _, c := range o.config.CommunityDescription.Chats {
		if c.CategoryId == chat.CategoryId {
			chat.Position++
		}
	}

	chat.Members = o.config.CommunityDescription.Members

	o.config.CommunityDescription.Chats[chatID] = chat

	return nil
}

func (o *Community) deleteChat(chatID string) *CommunityChanges {
	if o.config.CommunityDescription.Chats == nil {
		o.config.CommunityDescription.Chats = make(map[string]*protobuf.CommunityChat)
	}

	changes := o.emptyCommunityChanges()

	if chat, exists := o.config.CommunityDescription.Chats[chatID]; exists {
		tmpCatID := chat.CategoryId
		chat.CategoryId = ""
		o.SortCategoryChats(changes, tmpCatID)
		changes.ChatsRemoved[chatID] = chat
	}

	delete(o.config.CommunityDescription.Chats, chatID)
	return changes
}

func (o *Community) addCommunityMember(pk *ecdsa.PublicKey, member *protobuf.CommunityMember) {

	if o.config.CommunityDescription.Members == nil {
		o.config.CommunityDescription.Members = make(map[string]*protobuf.CommunityMember)
	}

	memberKey := common.PubkeyToHex(pk)
	o.config.CommunityDescription.Members[memberKey] = member
}

func (o *Community) addTokenPermission(permission *protobuf.CommunityTokenPermission) (*CommunityChanges, error) {
	if o.config.CommunityDescription.TokenPermissions == nil {
		o.config.CommunityDescription.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission)
	}

	if _, exists := o.config.CommunityDescription.TokenPermissions[permission.Id]; exists {
		return nil, ErrTokenPermissionAlreadyExists
	}

	o.config.CommunityDescription.TokenPermissions[permission.Id] = permission

	changes := o.emptyCommunityChanges()

	if changes.TokenPermissionsAdded == nil {
		changes.TokenPermissionsAdded = make(map[string]*protobuf.CommunityTokenPermission)
	}
	changes.TokenPermissionsAdded[permission.Id] = permission

	return changes, nil
}

func (o *Community) updateTokenPermission(permission *protobuf.CommunityTokenPermission) (*CommunityChanges, error) {
	if o.config.CommunityDescription.TokenPermissions == nil {
		o.config.CommunityDescription.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission)
	}
	if _, ok := o.config.CommunityDescription.TokenPermissions[permission.Id]; !ok {
		return nil, ErrTokenPermissionNotFound
	}

	changes := o.emptyCommunityChanges()
	o.config.CommunityDescription.TokenPermissions[permission.Id] = permission

	if changes.TokenPermissionsModified == nil {
		changes.TokenPermissionsModified = make(map[string]*protobuf.CommunityTokenPermission)
	}
	changes.TokenPermissionsModified[permission.Id] = o.config.CommunityDescription.TokenPermissions[permission.Id]

	return changes, nil
}

func (o *Community) deleteTokenPermission(permissionID string) (*CommunityChanges, error) {
	permission, exists := o.config.CommunityDescription.TokenPermissions[permissionID]
	if !exists {
		return nil, ErrTokenPermissionNotFound
	}

	delete(o.config.CommunityDescription.TokenPermissions, permissionID)

	changes := o.emptyCommunityChanges()

	changes.TokenPermissionsRemoved[permissionID] = permission
	return changes, nil
}

func (o *Community) addMemberWithRevealedAccounts(memberKey string, roles []protobuf.CommunityMember_Roles, accounts []*protobuf.RevealedAccount, clock uint64) *CommunityChanges {
	changes := o.emptyCommunityChanges()

	if o.config.CommunityDescription.Members == nil {
		o.config.CommunityDescription.Members = make(map[string]*protobuf.CommunityMember)
	}

	if _, ok := o.config.CommunityDescription.Members[memberKey]; !ok {
		o.config.CommunityDescription.Members[memberKey] = &protobuf.CommunityMember{Roles: roles}
		changes.MembersAdded[memberKey] = o.config.CommunityDescription.Members[memberKey]
	}

	o.config.CommunityDescription.Members[memberKey].RevealedAccounts = accounts
	o.config.CommunityDescription.Members[memberKey].LastUpdateClock = clock
	changes.MemberWalletsAdded[memberKey] = o.config.CommunityDescription.Members[memberKey].RevealedAccounts

	return changes
}

func (o *Community) DeclineRequestToJoin(dbRequest *RequestToJoin) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	isControlNode := o.IsControlNode()
	allowedToSendEvents := o.HasPermissionToSendCommunityEvents()

	if !isControlNode && !allowedToSendEvents {
		return ErrNotAdmin
	}

	if allowedToSendEvents {
		rejectedRequestsToJoin := make(map[string]*protobuf.CommunityRequestToJoin)
		rejectedRequestsToJoin[dbRequest.PublicKey] = dbRequest.ToCommunityRequestToJoinProtobuf()

		adminChanges := &CommunityEventChanges{
			CommunityChanges:       o.emptyCommunityChanges(),
			RejectedRequestsToJoin: rejectedRequestsToJoin,
		}
		err := o.addNewCommunityEvent(o.ToCommunityRequestToJoinRejectCommunityEvent(adminChanges))
		if err != nil {
			return err
		}
	}

	if isControlNode {
		// typically, community's clock is increased implicitly when making changes
		// to it, however in this scenario there are no changes in the community, yet
		// we need to increase the clock to ensure the owner event is processed by other
		// nodes.
		o.increaseClock()
	}

	return nil
}
