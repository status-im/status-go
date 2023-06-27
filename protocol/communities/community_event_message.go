package communities

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/status-im/status-go/protocol/protobuf"
)

type CommunityAdminEvent struct {
	AdminEventClock        uint64                                      `json:"clock"`
	Type                   protobuf.CommunityAdminEvent_EventType      `json:"type"`
	CommunityConfig        *protobuf.CommunityConfig                   `json:"community_config,omitempty"`
	TokenPermission        *protobuf.CommunityTokenPermission          `json:"token_permissions,omitempty"`
	CategoryData           *protobuf.CategoryData                      `json:"category_data,omitempty"`
	ChannelData            *protobuf.ChannelData                       `json:"channel_data,omitempty"`
	MemberToAction         string                                      `json:"member_to_action,omitempty"`
	MembersAdded           map[string]*protobuf.CommunityMember        `json:"membersAdded,omitempty"`
	RejectedRequestsToJoin map[string]*protobuf.CommunityRequestToJoin `json:"rejectedRequestsToJoin,omitempty"`
	AcceptedRequestsToJoin map[string]*protobuf.CommunityRequestToJoin `json:"acceptedRequestsToJoin,omitempty"`
	RawPayload             []byte                                      `json:"rawPayload"`
	Applied                bool                                        `json:"applied"`
}

func (e *CommunityAdminEvent) ToProtobuf() *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		AdminEventClock:        e.AdminEventClock,
		Type:                   e.Type,
		CommunityConfig:        e.CommunityConfig,
		TokenPermission:        e.TokenPermission,
		CategoryData:           e.CategoryData,
		ChannelData:            e.ChannelData,
		MemberToAction:         e.MemberToAction,
		MembersAdded:           e.MembersAdded,
		RejectedRequestsToJoin: e.RejectedRequestsToJoin,
		AcceptedRequestsToJoin: e.AcceptedRequestsToJoin,
	}
}

func CommunityAdminEventFromProtobuf(raw []byte) (*CommunityAdminEvent, error) {
	decodedEvent := protobuf.CommunityAdminEvent{}
	encodedEvent := raw

	err := proto.Unmarshal(encodedEvent, &decodedEvent)
	if err != nil {
		return nil, err
	}
	return &CommunityAdminEvent{
		AdminEventClock:        decodedEvent.AdminEventClock,
		Type:                   decodedEvent.Type,
		CommunityConfig:        decodedEvent.CommunityConfig,
		TokenPermission:        decodedEvent.TokenPermission,
		CategoryData:           decodedEvent.CategoryData,
		ChannelData:            decodedEvent.ChannelData,
		MemberToAction:         decodedEvent.MemberToAction,
		MembersAdded:           decodedEvent.MembersAdded,
		RejectedRequestsToJoin: decodedEvent.RejectedRequestsToJoin,
		AcceptedRequestsToJoin: decodedEvent.AcceptedRequestsToJoin,
		RawPayload:             encodedEvent,
	}, nil
}

type CommunityEventsMessage struct {
	CommunityId               []byte                `json:"community_id"`
	CommunityDescriptionClock uint64                `json:"community_description_clock"`
	AdminsEvents              []CommunityAdminEvent `json:"admins_events,omitempty"`
}

func (m *CommunityEventsMessage) ToProtobuf() protobuf.CommunityEventsMessage {
	rawEvents := adminsEventsToBytes(m.AdminsEvents)

	return protobuf.CommunityEventsMessage{
		CommunityId:               m.CommunityId,
		CommunityDescriptionClock: m.CommunityDescriptionClock,
		AdminsEvents:              rawEvents,
	}
}

func CommunityEventsMessageFromProtobuf(raw *protobuf.CommunityEventsMessage) (*CommunityEventsMessage, error) {
	events, err := adminsEventsFromBytes(raw.AdminsEvents)
	if err != nil {
		return nil, err
	}

	return &CommunityEventsMessage{
		CommunityId:               raw.CommunityId,
		CommunityDescriptionClock: raw.CommunityDescriptionClock,
		AdminsEvents:              events,
	}, nil
}

func (m *CommunityEventsMessage) Marshal() ([]byte, error) {
	pb := m.ToProtobuf()
	return proto.Marshal(&pb)
}

func (c *Community) mergeCommunityAdminEvents(src []CommunityAdminEvent) {
	for _, update := range src {
		var exists bool
		for _, existing := range c.config.AdminsEvents {
			if isCommunityAdminEventsEqual(&update, &existing) {
				exists = true
				break
			}
		}
		if !exists {
			c.config.AdminsEvents = append(c.config.AdminsEvents, update)
		}
	}

	c.sortCommunityAdminEvents()
}

func (c *Community) sortCommunityAdminEvents() {
	sort.Slice(c.config.AdminsEvents, func(i, j int) bool {
		return c.config.AdminsEvents[i].AdminEventClock < c.config.AdminsEvents[j].AdminEventClock
	})
}

func validateCommunityAdminEvent(adminEvent *CommunityAdminEvent) error {
	if adminEvent.Type == protobuf.CommunityAdminEvent_UNKNOWN {
		return errors.New("unknown admin event")
	}

	switch adminEvent.Type {
	case protobuf.CommunityAdminEvent_COMMUNITY_EDIT:
		if adminEvent.CommunityConfig == nil || adminEvent.CommunityConfig.Identity == nil ||
			adminEvent.CommunityConfig.Permissions == nil || adminEvent.CommunityConfig.AdminSettings == nil {
			return errors.New("invalid config change admin event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_CHANGE:
		if adminEvent.TokenPermission == nil || len(adminEvent.TokenPermission.Id) == 0 {
			return errors.New("invalid token permission change event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETE:
		if adminEvent.TokenPermission == nil || len(adminEvent.TokenPermission.Id) == 0 {
			return errors.New("invalid token permission delete event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CREATE:
		if adminEvent.CategoryData == nil || len(adminEvent.CategoryData.CategoryId) == 0 {
			return errors.New("invalid community category create event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_DELETE:
		if adminEvent.CategoryData == nil || len(adminEvent.CategoryData.CategoryId) == 0 {
			return errors.New("invalid community category delete event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_EDIT:
		if adminEvent.CategoryData == nil || len(adminEvent.CategoryData.CategoryId) == 0 {
			return errors.New("invalid community category edit event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CREATE:
		if adminEvent.ChannelData == nil || len(adminEvent.ChannelData.ChannelId) == 0 ||
			adminEvent.ChannelData.Channel == nil {
			return errors.New("invalid community channel create event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_DELETE:
		if adminEvent.ChannelData == nil || len(adminEvent.ChannelData.ChannelId) == 0 {
			return errors.New("invalid community channel delete event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_EDIT:
		if adminEvent.ChannelData == nil || len(adminEvent.ChannelData.ChannelId) == 0 ||
			adminEvent.ChannelData.Channel == nil {
			return errors.New("invalid community channel edit event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_REORDER:
		if adminEvent.ChannelData == nil || len(adminEvent.ChannelData.ChannelId) == 0 {
			return errors.New("invalid community channel reorder event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_REORDER:
		if adminEvent.CategoryData == nil || len(adminEvent.CategoryData.CategoryId) == 0 {
			return errors.New("invalid community category reorder event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT:
		if len(adminEvent.MembersAdded) == 0 {
			return errors.New("invalid community request to join accepted event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT:
		if adminEvent.RejectedRequestsToJoin == nil {
			return errors.New("invalid community request to join reject event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_KICK:
		if len(adminEvent.MemberToAction) == 0 {
			return errors.New("invalid community member kick event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_BAN:
		if len(adminEvent.MemberToAction) == 0 {
			return errors.New("invalid community member ban event")
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_UNBAN:
		if len(adminEvent.MemberToAction) == 0 {
			return errors.New("invalid community member unban event")
		}

	default:
		return errors.New("unknown admin community event")
	}
	return nil
}

func isCommunityAdminEventsEqual(left *CommunityAdminEvent, right *CommunityAdminEvent) bool {
	return bytes.Equal(left.RawPayload, right.RawPayload)
}

func adminsEventsToBytes(adminEvents []CommunityAdminEvent) [][]byte {
	var rawEvents [][]byte
	for _, e := range adminEvents {
		var encodedEvent []byte
		encodedEvent = append(encodedEvent, e.RawPayload...)
		rawEvents = append(rawEvents, encodedEvent)
	}

	return rawEvents
}

func adminsEventsFromBytes(rawEvents [][]byte) ([]CommunityAdminEvent, error) {
	var events []CommunityAdminEvent
	for _, e := range rawEvents {
		verifiedEvent, err := CommunityAdminEventFromProtobuf(e)
		if err != nil {
			return nil, err
		}
		events = append(events, *verifiedEvent)
	}
	return events, nil
}

func adminsEventsToJsonEncodedBytes(adminEvents []CommunityAdminEvent) ([]byte, error) {
	return json.Marshal(adminEvents)
}

func adminsEventsFromJsonEncodedBytes(jsonEncodedRawEvents []byte) ([]CommunityAdminEvent, error) {
	var events []CommunityAdminEvent
	err := json.Unmarshal(jsonEncodedRawEvents, &events)
	if err != nil {
		return nil, err
	}

	return events, nil
}
