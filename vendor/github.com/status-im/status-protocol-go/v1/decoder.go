package statusproto

import (
	"container/list"
	"errors"
	"fmt"
	"io"

	"github.com/russolsen/transit"
)

// NewMessageDecoder returns a new Transit decoder
// that can deserialize Message structs.
// More about Transit: https://github.com/cognitect/transit-format
func NewMessageDecoder(r io.Reader) *transit.Decoder {
	decoder := transit.NewDecoder(r)
	decoder.AddHandler(pairMessageTag, pairMessageHandler)
	decoder.AddHandler(membershipUpdateTag, membershipUpdateMessageHandler)
	return decoder
}

const (
	messageTag          = "c4"
	pairMessageTag      = "p2"
	membershipUpdateTag = "g5"
)

func pairMessageHandler(d transit.Decoder, value interface{}) (interface{}, error) {
	taggedValue, ok := value.(transit.TaggedValue)
	if !ok {
		return nil, errors.New("not a tagged value")
	}
	values, ok := taggedValue.Value.([]interface{})
	if !ok {
		return nil, errors.New("tagged value does not contain values")
	}

	pm := PairMessage{}
	for idx, v := range values {
		var ok bool

		switch idx {
		case 0:
			pm.InstallationID, ok = v.(string)
		case 1:
			pm.DeviceType, ok = v.(string)
		case 2:
			pm.Name, ok = v.(string)
		case 3:
			pm.FCMToken, ok = v.(string)
		default:
			// skip any other values
			ok = true
		}

		if !ok {
			return nil, fmt.Errorf("invalid value for index: %d", idx)
		}
	}
	return pm, nil
}

func membershipUpdateMessageHandler(d transit.Decoder, value interface{}) (interface{}, error) {
	taggedValue, ok := value.(transit.TaggedValue)
	if !ok {
		return nil, errors.New("not a tagged value")
	}
	values, ok := taggedValue.Value.([]interface{})
	if !ok {
		return nil, errors.New("tagged value does not contain values")
	}

	m := MembershipUpdateMessage{}
	for idx, v := range values {
		var ok bool

		switch idx {
		case 0:
			m.ChatID, ok = v.(string)
		case 1:
			var updates *list.List
			updates, ok = v.(*list.List)
			if !ok {
				break
			}
			for e := updates.Front(); e != nil; e = e.Next() {
				var value map[interface{}]interface{}
				value, ok = e.Value.(map[interface{}]interface{})
				if !ok {
					break
				}

				update := MembershipUpdate{}

				update.ChatID, ok = value[transit.Keyword("chat-id")].(string)
				if !ok {
					break
				}
				update.Signature, ok = value[transit.Keyword("signature")].(string)
				if !ok {
					break
				}

				// parse events
				var events []interface{}
				events, ok = value[transit.Keyword("events")].([]interface{})
				if !ok {
					break
				}
				for _, item := range events {
					var event map[interface{}]interface{}
					event, ok = item.(map[interface{}]interface{})
					if !ok {
						break
					}

					var updateEvent MembershipUpdateEvent
					updateEvent, ok = parseEvent(event)
					if !ok {
						break
					}

					update.Events = append(update.Events, updateEvent)
				}

				m.Updates = append(m.Updates, update)
			}
		default:
			// skip any other values
			ok = true
		}

		if !ok {
			return nil, fmt.Errorf("invalid value for index: %d", idx)
		}
	}
	return m, nil
}

func setToString(set *transit.Set) ([]string, bool) {
	result := make([]string, 0, len(set.Contents))
	for _, item := range set.Contents {
		val, ok := item.(string)
		if !ok {
			return nil, false
		}
		result = append(result, val)
	}
	return result, true
}

func parseEvent(event map[interface{}]interface{}) (result MembershipUpdateEvent, ok bool) {
	// Type is required
	result.Type, ok = event[transit.Keyword("type")].(string)
	if !ok {
		return
	}
	// ClockValue is required
	result.ClockValue, ok = event[transit.Keyword("clock-value")].(int64)
	if !ok {
		return
	}
	// Name is optional
	if val, exists := event[transit.Keyword("name")]; exists {
		result.Name, ok = val.(string)
		if !ok {
			return
		}
	}
	// Member is optional
	if val, exists := event[transit.Keyword("member")]; exists {
		result.Member, ok = val.(string)
		if !ok {
			return
		}
	}
	// Members is optional
	if val, exists := event[transit.Keyword("members")]; exists {
		var members *transit.Set
		members, ok = val.(*transit.Set)
		if !ok {
			return
		}
		result.Members, ok = setToString(members)
		if !ok {
			return
		}
	}
	return
}
