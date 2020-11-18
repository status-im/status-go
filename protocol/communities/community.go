package communities

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"sync"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
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
	Verified                      bool
	Logger                        *zap.Logger
}

type Community struct {
	config *Config
	mutex  sync.Mutex
}

func New(config Config) (*Community, error) {
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

func (o *Community) MarshalJSON() ([]byte, error) {
	item := struct {
		*protobuf.CommunityDescription `json:"description"`
		ID                             string `json:"id"`
		Admin                          bool   `json:"admin"`
		Verified                       bool   `json:"verified"`
		Joined                         bool   `json:"joined"`
	}{
		ID:                   o.IDString(),
		CommunityDescription: o.config.CommunityDescription,
		Admin:                o.IsAdmin(),
		Verified:             o.config.Verified,
		Joined:               o.config.Joined,
	}
	return json.Marshal(item)
}

func (o *Community) initialize() {
	if o.config.CommunityDescription == nil {
		o.config.CommunityDescription = &protobuf.CommunityDescription{}

	}
}

type CommunityChatChanges struct {
	MembersAdded   map[string]*protobuf.CommunityMember
	MembersRemoved map[string]*protobuf.CommunityMember
}

type CommunityChanges struct {
	MembersAdded   map[string]*protobuf.CommunityMember
	MembersRemoved map[string]*protobuf.CommunityMember

	ChatsRemoved  map[string]*protobuf.CommunityChat
	ChatsAdded    map[string]*protobuf.CommunityChat
	ChatsModified map[string]*CommunityChatChanges
}

func emptyCommunityChanges() *CommunityChanges {
	return &CommunityChanges{
		MembersAdded:   make(map[string]*protobuf.CommunityMember),
		MembersRemoved: make(map[string]*protobuf.CommunityMember),

		ChatsRemoved:  make(map[string]*protobuf.CommunityChat),
		ChatsAdded:    make(map[string]*protobuf.CommunityChat),
		ChatsModified: make(map[string]*CommunityChatChanges),
	}
}

func (o *Community) CreateChat(chatID string, chat *protobuf.CommunityChat) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
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

	o.config.CommunityDescription.Chats[chatID] = chat

	o.increaseClock()

	changes := emptyCommunityChanges()
	changes.ChatsAdded[chatID] = chat
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

func (o *Community) hasMember(pk *ecdsa.PublicKey) bool {

	key := common.PubkeyToHex(pk)
	_, ok := o.config.CommunityDescription.Members[key]
	return ok
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

	return o.config.CommunityDescription, nil
}

func (o *Community) AcceptRequestToJoin(pk *ecdsa.PublicKey) (*protobuf.CommunityRequestJoinResponse, error) {

	return nil, nil
}

func (o *Community) DeclineRequestToJoin(pk *ecdsa.PublicKey) (*protobuf.CommunityRequestJoinResponse, error) {
	return nil, nil
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

func (o *Community) HandleCommunityDescription(signer *ecdsa.PublicKey, description *protobuf.CommunityDescription, rawMessage []byte) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !common.IsPubKeyEqual(o.config.ID, signer) {
		return nil, ErrNotAuthorized
	}

	err := ValidateCommunityDescription(description)
	if err != nil {
		return nil, err
	}

	response := emptyCommunityChanges()

	if description.Clock <= o.config.CommunityDescription.Clock {
		return response, nil
	}

	// We only calculate changes if we joined the org, otherwise not interested
	if o.config.Joined {
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
	}

	o.config.CommunityDescription = description
	o.config.MarshaledCommunityDescription = rawMessage

	return response, nil
}

// HandleRequestJoin handles a request, checks that the right permissions are applied and returns an CommunityRequestJoinResponse
func (o *Community) HandleRequestJoin(signer *ecdsa.PublicKey, request *protobuf.CommunityRequestJoin) error {
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
		return o.handleRequestJoinWithChatID(signer, request)
	}

	err := o.handleRequestJoinWithoutChatID(signer, request)
	if err != nil {
		return err
	}

	// Store request to join
	return nil
}

func (o *Community) IsAdmin() bool {
	return o.config.PrivateKey != nil
}

func (o *Community) handleRequestJoinWithChatID(signer *ecdsa.PublicKey, request *protobuf.CommunityRequestJoin) error {

	var chat *protobuf.CommunityChat
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

func (o *Community) handleRequestJoinWithoutChatID(signer *ecdsa.PublicKey, request *protobuf.CommunityRequestJoin) error {

	// If they want access to the org only, check that the org is ON_REQUEST
	if o.config.CommunityDescription.Permissions.Access != protobuf.CommunityPermissions_ON_REQUEST {
		return ErrCantRequestAccess
	}

	return nil
}

func (o *Community) ID() []byte {
	return crypto.CompressPubkey(o.config.ID)
}

func (o *Community) IDString() string {
	return types.EncodeHex(o.ID())
}

func (o *Community) PrivateKey() *ecdsa.PrivateKey {
	return o.config.PrivateKey
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

func (o *Community) nextClock() uint64 {
	return o.config.CommunityDescription.Clock + 1
}
