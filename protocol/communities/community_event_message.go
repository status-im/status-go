package communities

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/status-im/status-go/protocol/protobuf"
)

type CommunityEvent struct {
	CommunityEventClock    uint64                                      `json:"clock"`
	Type                   protobuf.CommunityEvent_EventType           `json:"type"`
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

func (e *CommunityEvent) ToProtobuf() *protobuf.CommunityEvent {
	return &protobuf.CommunityEvent{
		CommunityEventClock:    e.CommunityEventClock,
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

func CommunityEventFromProtobuf(raw []byte) (*CommunityEvent, error) {
	decodedEvent := protobuf.CommunityEvent{}
	encodedEvent := raw

	err := proto.Unmarshal(encodedEvent, &decodedEvent)
	if err != nil {
		return nil, err
	}
	return &CommunityEvent{
		CommunityEventClock:    decodedEvent.CommunityEventClock,
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
	CommunityID               []byte           `json:"community_id"`
	CommunityDescriptionClock uint64           `json:"community_description_clock"`
	Events                    []CommunityEvent `json:"admins_events,omitempty"`
}

func (m *CommunityEventsMessage) ToProtobuf() protobuf.CommunityEventsMessage {
	rawEvents := communityEventsToBytes(m.Events)

	return protobuf.CommunityEventsMessage{
		CommunityId:               m.CommunityID,
		CommunityDescriptionClock: m.CommunityDescriptionClock,
		Events:                    rawEvents,
	}
}

func CommunityEventsMessageFromProtobuf(raw *protobuf.CommunityEventsMessage) (*CommunityEventsMessage, error) {
	events, err := communityEventsFromBytes(raw.Events)
	if err != nil {
		return nil, err
	}

	return &CommunityEventsMessage{
		CommunityID:               raw.CommunityId,
		CommunityDescriptionClock: raw.CommunityDescriptionClock,
		Events:                    events,
	}, nil
}

func (m *CommunityEventsMessage) Marshal() ([]byte, error) {
	pb := m.ToProtobuf()
	return proto.Marshal(&pb)
}

func (c *Community) mergeCommunityEvents(src []CommunityEvent) {
	for _, update := range src {
		var exists bool
		for _, existing := range c.config.Events {
			if isCommunityEventsEqual(update, existing) {
				exists = true
				break
			}
		}
		if !exists {
			c.config.Events = append(c.config.Events, update)
		}
	}

	c.sortCommunityEvents()
}

func (c *Community) sortCommunityEvents() {
	sort.Slice(c.config.Events, func(i, j int) bool {
		return c.config.Events[i].CommunityEventClock < c.config.Events[j].CommunityEventClock
	})
}

func validateCommunityEvent(communityEvent *CommunityEvent) error {
	if communityEvent.Type == protobuf.CommunityEvent_UNKNOWN {
		return errors.New("unknown admin event")
	}

	switch communityEvent.Type {
	case protobuf.CommunityEvent_COMMUNITY_EDIT:
		if communityEvent.CommunityConfig == nil || communityEvent.CommunityConfig.Identity == nil ||
			communityEvent.CommunityConfig.Permissions == nil || communityEvent.CommunityConfig.AdminSettings == nil {
			return errors.New("invalid config change admin event")
		}

	case protobuf.CommunityEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_CHANGE:
		if communityEvent.TokenPermission == nil || len(communityEvent.TokenPermission.Id) == 0 {
			return errors.New("invalid token permission change event")
		}

	case protobuf.CommunityEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETE:
		if communityEvent.TokenPermission == nil || len(communityEvent.TokenPermission.Id) == 0 {
			return errors.New("invalid token permission delete event")
		}

	case protobuf.CommunityEvent_COMMUNITY_CATEGORY_CREATE:
		if communityEvent.CategoryData == nil || len(communityEvent.CategoryData.CategoryId) == 0 {
			return errors.New("invalid community category create event")
		}

	case protobuf.CommunityEvent_COMMUNITY_CATEGORY_DELETE:
		if communityEvent.CategoryData == nil || len(communityEvent.CategoryData.CategoryId) == 0 {
			return errors.New("invalid community category delete event")
		}

	case protobuf.CommunityEvent_COMMUNITY_CATEGORY_EDIT:
		if communityEvent.CategoryData == nil || len(communityEvent.CategoryData.CategoryId) == 0 {
			return errors.New("invalid community category edit event")
		}

	case protobuf.CommunityEvent_COMMUNITY_CHANNEL_CREATE:
		if communityEvent.ChannelData == nil || len(communityEvent.ChannelData.ChannelId) == 0 ||
			communityEvent.ChannelData.Channel == nil {
			return errors.New("invalid community channel create event")
		}

	case protobuf.CommunityEvent_COMMUNITY_CHANNEL_DELETE:
		if communityEvent.ChannelData == nil || len(communityEvent.ChannelData.ChannelId) == 0 {
			return errors.New("invalid community channel delete event")
		}

	case protobuf.CommunityEvent_COMMUNITY_CHANNEL_EDIT:
		if communityEvent.ChannelData == nil || len(communityEvent.ChannelData.ChannelId) == 0 ||
			communityEvent.ChannelData.Channel == nil {
			return errors.New("invalid community channel edit event")
		}

	case protobuf.CommunityEvent_COMMUNITY_CHANNEL_REORDER:
		if communityEvent.ChannelData == nil || len(communityEvent.ChannelData.ChannelId) == 0 {
			return errors.New("invalid community channel reorder event")
		}

	case protobuf.CommunityEvent_COMMUNITY_CATEGORY_REORDER:
		if communityEvent.CategoryData == nil || len(communityEvent.CategoryData.CategoryId) == 0 {
			return errors.New("invalid community category reorder event")
		}

	case protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT:
		if len(communityEvent.MembersAdded) == 0 {
			return errors.New("invalid community request to join accepted event")
		}

	case protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT:
		if communityEvent.RejectedRequestsToJoin == nil {
			return errors.New("invalid community request to join reject event")
		}

	case protobuf.CommunityEvent_COMMUNITY_MEMBER_KICK:
		if len(communityEvent.MemberToAction) == 0 {
			return errors.New("invalid community member kick event")
		}

	case protobuf.CommunityEvent_COMMUNITY_MEMBER_BAN:
		if len(communityEvent.MemberToAction) == 0 {
			return errors.New("invalid community member ban event")
		}

	case protobuf.CommunityEvent_COMMUNITY_MEMBER_UNBAN:
		if len(communityEvent.MemberToAction) == 0 {
			return errors.New("invalid community member unban event")
		}

	default:
		return errors.New("unknown admin community event")
	}
	return nil
}

func isCommunityEventsEqual(left CommunityEvent, right CommunityEvent) bool {
	return bytes.Equal(left.RawPayload, right.RawPayload)
}

func communityEventsToBytes(communityEvents []CommunityEvent) [][]byte {
	var rawEvents [][]byte
	for _, e := range communityEvents {
		var encodedEvent []byte
		encodedEvent = append(encodedEvent, e.RawPayload...)
		rawEvents = append(rawEvents, encodedEvent)
	}

	return rawEvents
}

func communityEventsFromBytes(rawEvents [][]byte) ([]CommunityEvent, error) {
	var events []CommunityEvent
	for _, e := range rawEvents {
		verifiedEvent, err := CommunityEventFromProtobuf(e)
		if err != nil {
			return nil, err
		}
		events = append(events, *verifiedEvent)
	}
	return events, nil
}

func communityEventsToJSONEncodedBytes(communityEvents []CommunityEvent) ([]byte, error) {
	return json.Marshal(communityEvents)
}

func communityEventsFromJSONEncodedBytes(jsonEncodedRawEvents []byte) ([]CommunityEvent, error) {
	var events []CommunityEvent
	err := json.Unmarshal(jsonEncodedRawEvents, &events)
	if err != nil {
		return nil, err
	}

	return events, nil
}
