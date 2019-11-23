package protocol

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
)

const (
	MembershipUpdateChatCreated   = "chat-created"
	MembershipUpdateNameChanged   = "name-changed"
	MembershipUpdateMembersAdded  = "members-added"
	MembershipUpdateMemberJoined  = "member-joined"
	MembershipUpdateMemberRemoved = "member-removed"
	MembershipUpdateAdminsAdded   = "admins-added"
	MembershipUpdateAdminRemoved  = "admin-removed"
)

// MembershipUpdateMessage is a message used to propagate information
// about group membership changes.
// For more information, see https://github.com/status-im/specs/blob/master/status-group-chats-spec.md.
type MembershipUpdateMessage struct {
	ChatID  string             `json:"chatId"` // UUID concatenated with hex-encoded public key of the creator for the chat
	Updates []MembershipUpdate `json:"updates"`
	Message *Message           `json:"message"` // optional message
}

// Verify makes sure that the received update message has a valid signature.
// It also extracts public key from the signature available as From field.
// It does not verify the updates and their events. This should be done
// separately using Group struct.
func (m *MembershipUpdateMessage) Verify() error {
	for idx, update := range m.Updates {
		if err := update.extractFrom(); err != nil {
			return errors.Wrapf(err, "failed to extract an author of %d update", idx)
		}
		m.Updates[idx] = update
	}
	return nil
}

// EncodeMembershipUpdateMessage encodes a MembershipUpdateMessage using Transit serialization.
func EncodeMembershipUpdateMessage(value MembershipUpdateMessage) ([]byte, error) {
	var buf bytes.Buffer
	encoder := NewMessageEncoder(&buf)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type MembershipUpdate struct {
	ChatID    string                  `json:"chatId"`
	Signature string                  `json:"signature"` // hex-encoded without 0x prefix
	Events    []MembershipUpdateEvent `json:"events"`
	From      string                  `json:"from"` // hex-encoded with 0x prefix
}

// Sign creates a signature from MembershipUpdateEvents
// and updates MembershipUpdate's signature.
// It follows the algorithm describe in the spec:
// https://github.com/status-im/specs/blob/master/status-group-chats-spec.md#signature.
func (u *MembershipUpdate) Sign(identity *ecdsa.PrivateKey) error {
	signature, err := createMembershipUpdateSignature(u.ChatID, u.Events, identity)
	if err != nil {
		return err
	}
	u.Signature = signature
	return nil
}

func (u *MembershipUpdate) extractFrom() error {
	content, err := stringifyMembershipUpdateEvents(u.ChatID, u.Events)
	if err != nil {
		return errors.Wrap(err, "failed to stringify events")
	}
	signatureBytes, err := hex.DecodeString(u.Signature)
	if err != nil {
		return errors.Wrap(err, "failed to decode signature")
	}
	publicKey, err := crypto.ExtractSignature(content, signatureBytes)
	if err != nil {
		return errors.Wrap(err, "failed to extract signature")
	}
	u.From = types.EncodeHex(crypto.FromECDSAPub(publicKey))
	return nil
}

func (u *MembershipUpdate) Flat() []MembershipUpdateFlat {
	result := make([]MembershipUpdateFlat, 0, len(u.Events))
	for _, event := range u.Events {
		result = append(result, MembershipUpdateFlat{
			MembershipUpdateEvent: event,
			ChatID:                u.ChatID,
			Signature:             u.Signature,
			From:                  u.From,
		})
	}
	return result
}

// MembershipUpdateEvent contains an event information.
// Member and Members are hex-encoded values with 0x prefix.
type MembershipUpdateEvent struct {
	Type       string   `json:"type"`
	ClockValue int64    `json:"clockValue"`
	Member     string   `json:"member,omitempty"`  // in "member-joined", "member-removed" and "admin-removed" events
	Members    []string `json:"members,omitempty"` // in "members-added" and "admins-added" events
	Name       string   `json:"name,omitempty"`    // name of the group chat
}

func (u MembershipUpdateEvent) Equal(update MembershipUpdateEvent) bool {
	return u.Type == update.Type &&
		u.ClockValue == update.ClockValue &&
		u.Member == update.Member &&
		stringSliceEquals(u.Members, update.Members) &&
		u.Name == update.Name
}

type MembershipUpdateFlat struct {
	MembershipUpdateEvent
	ChatID    string `json:"chatId"`
	Signature string `json:"signature"`
	From      string `json:"from"`
}

func (u MembershipUpdateFlat) Equal(update MembershipUpdateFlat) bool {
	return u.ChatID == update.ChatID &&
		u.Signature == update.Signature &&
		u.From == update.From &&
		u.MembershipUpdateEvent.Equal(update.MembershipUpdateEvent)
}

func MergeFlatMembershipUpdates(dest []MembershipUpdateFlat, src []MembershipUpdateFlat) []MembershipUpdateFlat {
	for _, update := range src {
		var exists bool
		for _, existing := range dest {
			if existing.Equal(update) {
				exists = true
				break
			}
		}
		if !exists {
			dest = append(dest, update)
		}
	}
	return dest
}

func NewChatCreatedEvent(name string, admin string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateChatCreated,
		Name:       name,
		Member:     admin,
		ClockValue: clock,
	}
}

func NewNameChangedEvent(name string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateNameChanged,
		Name:       name,
		ClockValue: clock,
	}
}

func NewMembersAddedEvent(members []string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateMembersAdded,
		Members:    members,
		ClockValue: clock,
	}
}

func NewMemberJoinedEvent(member string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateMemberJoined,
		Member:     member,
		ClockValue: clock,
	}
}

func NewAdminsAddedEvent(admins []string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateAdminsAdded,
		Members:    admins,
		ClockValue: clock,
	}
}

func NewMemberRemovedEvent(member string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateMemberRemoved,
		Member:     member,
		ClockValue: clock,
	}
}

func NewAdminRemovedEvent(admin string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateAdminRemoved,
		Member:     admin,
		ClockValue: clock,
	}
}

func stringifyMembershipUpdateEvents(chatID string, events []MembershipUpdateEvent) ([]byte, error) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].ClockValue < events[j].ClockValue
	})
	tuples := make([]interface{}, len(events))
	for idx, event := range events {
		tuples[idx] = tupleMembershipUpdateEvent(event)
	}
	structureToSign := []interface{}{
		tuples,
		chatID,
	}
	return json.Marshal(structureToSign)
}

func createMembershipUpdateSignature(chatID string, events []MembershipUpdateEvent, identity *ecdsa.PrivateKey) (string, error) {
	data, err := stringifyMembershipUpdateEvents(chatID, events)
	if err != nil {
		return "", err
	}
	return crypto.SignBytesAsHex(data, identity)
}

var membershipUpdateEventFieldNamesCompat = map[string]string{
	"ClockValue": "clock-value",
	"Name":       "name",
	"Type":       "type",
	"Member":     "member",
	"Members":    "members",
}

func tupleMembershipUpdateEvent(update MembershipUpdateEvent) [][]interface{} {
	// Sort all slices first.
	sort.Slice(update.Members, func(i, j int) bool {
		return update.Members[i] < update.Members[j]
	})
	v := reflect.ValueOf(update)
	result := make([][]interface{}, 0, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		fieldName := v.Type().Field(i).Name
		if name, exists := membershipUpdateEventFieldNamesCompat[fieldName]; exists {
			fieldName = name
		}
		field := v.Field(i)
		if !isZeroValue(field) {
			result = append(result, []interface{}{fieldName, field.Interface()})
		}
	}
	// Sort the result lexicographically.
	// We know that the first item of a tuple is a string
	// because it's a field name.
	sort.Slice(result, func(i, j int) bool {
		return result[i][0].(string) < result[j][0].(string)
	})
	return result
}

type Group struct {
	chatID  string
	name    string
	updates []MembershipUpdateFlat
	admins  *stringSet
	members *stringSet
}

func groupChatID(creator *ecdsa.PublicKey) string {
	return uuid.New().String() + "-" + types.EncodeHex(crypto.FromECDSAPub(creator))
}

func NewGroupWithMembershipUpdates(chatID string, updates []MembershipUpdate) (*Group, error) {
	flatten := make([]MembershipUpdateFlat, 0, len(updates))
	for _, update := range updates {
		flatten = append(flatten, update.Flat()...)
	}
	return newGroup(chatID, flatten)
}

func NewGroupWithCreator(name string, creator *ecdsa.PrivateKey) (*Group, error) {
	chatID := groupChatID(&creator.PublicKey)
	creatorHex := publicKeyToString(&creator.PublicKey)
	clock := TimestampInMsFromTime(time.Now())
	chatCreated := NewChatCreatedEvent(name, creatorHex, int64(clock))
	update := MembershipUpdate{
		ChatID: chatID,
		From:   creatorHex,
		Events: []MembershipUpdateEvent{chatCreated},
	}
	if err := update.Sign(creator); err != nil {
		return nil, err
	}
	return newGroup(chatID, update.Flat())
}

func NewGroup(chatID string, updates []MembershipUpdateFlat) (*Group, error) {
	return newGroup(chatID, updates)
}

func newGroup(chatID string, updates []MembershipUpdateFlat) (*Group, error) {
	g := Group{
		chatID:  chatID,
		updates: updates,
		admins:  newStringSet(),
		members: newStringSet(),
	}
	if err := g.init(); err != nil {
		return nil, err
	}
	return &g, nil
}

func (g *Group) init() error {
	g.sortEvents()

	var chatID string

	for _, update := range g.updates {
		if chatID == "" {
			chatID = update.ChatID
		} else if update.ChatID != chatID {
			return errors.New("updates contain different chat IDs")
		}
		valid := g.validateEvent(update.From, update.MembershipUpdateEvent)
		if !valid {
			return fmt.Errorf("invalid event %#+v from %s", update.MembershipUpdateEvent, update.From)
		}
		g.processEvent(update.From, update.MembershipUpdateEvent)
	}

	valid := g.validateChatID(g.chatID)
	if !valid {
		return fmt.Errorf("invalid chat ID: %s", g.chatID)
	}
	if chatID != g.chatID {
		return fmt.Errorf("expected chat ID equal %s, got %s", g.chatID, chatID)
	}

	return nil
}

func (g Group) ChatID() string {
	return g.chatID
}

func (g Group) Updates() []MembershipUpdateFlat {
	return g.updates
}

func (g Group) Name() string {
	return g.name
}

func (g Group) Members() []string {
	return g.members.List()
}

func (g Group) Admins() []string {
	return g.admins.List()
}

func (g Group) Joined() []string {
	var result []string
	for _, update := range g.updates {
		if update.Type == MembershipUpdateMemberJoined {
			result = append(result, update.Member)
		}
	}
	return result
}

func (g *Group) ProcessEvents(from *ecdsa.PublicKey, events []MembershipUpdateEvent) error {
	for _, event := range events {
		err := g.ProcessEvent(from, event)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Group) ProcessEvent(from *ecdsa.PublicKey, event MembershipUpdateEvent) error {
	fromHex := types.EncodeHex(crypto.FromECDSAPub(from))
	if !g.validateEvent(fromHex, event) {
		return fmt.Errorf("invalid event %#+v from %s", event, from)
	}
	update := MembershipUpdate{
		ChatID: g.chatID,
		From:   fromHex,
		Events: []MembershipUpdateEvent{event},
	}
	g.updates = append(g.updates, update.Flat()...)
	g.processEvent(fromHex, event)
	return nil
}

func (g Group) LastClockValue() int64 {
	if len(g.updates) == 0 {
		return 0
	}
	return g.updates[len(g.updates)-1].ClockValue
}

func (g Group) NextClockValue() int64 {
	return g.LastClockValue() + 1
}

func (g Group) creator() (string, error) {
	if len(g.updates) == 0 {
		return "", errors.New("no events in the group")
	}
	first := g.updates[0]
	if first.Type != MembershipUpdateChatCreated {
		return "", fmt.Errorf("expected first event to be 'chat-created', got %s", first.Type)
	}
	return first.From, nil
}

func (g Group) validateChatID(chatID string) bool {
	creator, err := g.creator()
	if err != nil || creator == "" {
		return false
	}
	// TODO: It does not verify that the prefix is a valid UUID.
	//       Improve it so that the prefix follows UUIDv4 spec.
	return strings.HasSuffix(chatID, creator) && chatID != creator
}

// validateEvent returns true if a given event is valid.
func (g Group) validateEvent(from string, event MembershipUpdateEvent) bool {
	switch event.Type {
	case MembershipUpdateChatCreated:
		return g.admins.Empty() && g.members.Empty()
	case MembershipUpdateNameChanged:
		return g.admins.Has(from) && len(event.Name) > 0
	case MembershipUpdateMembersAdded:
		return g.admins.Has(from)
	case MembershipUpdateMemberJoined:
		return g.members.Has(from) && from == event.Member
	case MembershipUpdateMemberRemoved:
		// Member can remove themselves or admin can remove a member.
		return from == event.Member || (g.admins.Has(from) && !g.admins.Has(event.Member))
	case MembershipUpdateAdminsAdded:
		return g.admins.Has(from) && stringSliceSubset(event.Members, g.members.List())
	case MembershipUpdateAdminRemoved:
		return g.admins.Has(from) && from == event.Member
	default:
		return false
	}
}

func (g *Group) processEvent(from string, event MembershipUpdateEvent) {
	switch event.Type {
	case MembershipUpdateChatCreated:
		g.name = event.Name
		g.members.Add(event.Member)
		g.admins.Add(event.Member)
	case MembershipUpdateNameChanged:
		g.name = event.Name
	case MembershipUpdateAdminsAdded:
		g.admins.Add(event.Members...)
	case MembershipUpdateAdminRemoved:
		g.admins.Remove(event.Member)
	case MembershipUpdateMembersAdded:
		g.members.Add(event.Members...)
	case MembershipUpdateMemberRemoved:
		g.members.Remove(event.Member)
	case MembershipUpdateMemberJoined:
		g.members.Add(event.Member)
	}
}

func (g *Group) sortEvents() {
	sort.Slice(g.updates, func(i, j int) bool {
		return g.updates[i].ClockValue < g.updates[j].ClockValue
	})
}

func stringSliceSubset(subset []string, set []string) bool {
	for _, item1 := range set {
		var found bool
		for _, item2 := range subset {
			if item1 == item2 {
				found = true
				break
			}
		}
		if found {
			return true
		}
	}
	return false
}

func stringSliceEquals(slice1, slice2 []string) bool {
	set := map[string]struct{}{}
	for _, s := range slice1 {
		set[s] = struct{}{}
	}
	for _, s := range slice2 {
		_, ok := set[s]
		if !ok {
			return false
		}
	}
	return true
}

func publicKeyToString(publicKey *ecdsa.PublicKey) string {
	return types.EncodeHex(crypto.FromECDSAPub(publicKey))
}

type stringSet struct {
	m     map[string]struct{}
	items []string
}

func newStringSet() *stringSet {
	return &stringSet{
		m: make(map[string]struct{}),
	}
}

func newStringSetFromSlice(s []string) *stringSet {
	set := newStringSet()
	if len(s) > 0 {
		set.Add(s...)
	}
	return set
}

func (s *stringSet) Add(items ...string) {
	for _, item := range items {
		if _, ok := s.m[item]; !ok {
			s.m[item] = struct{}{}
			s.items = append(s.items, item)
		}
	}
}

func (s *stringSet) Remove(items ...string) {
	for _, item := range items {
		if _, ok := s.m[item]; ok {
			delete(s.m, item)
			s.removeFromItems(item)
		}
	}
}

func (s *stringSet) Has(item string) bool {
	_, ok := s.m[item]
	return ok
}

func (s *stringSet) Empty() bool {
	return len(s.items) == 0
}

func (s *stringSet) List() []string {
	return s.items
}

func (s *stringSet) removeFromItems(dropped string) {
	n := 0
	for _, item := range s.items {
		if item != dropped {
			s.items[n] = item
			n++
		}
	}
	s.items = s.items[:n]
}
