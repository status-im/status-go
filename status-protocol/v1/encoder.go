package statusproto

import (
	"container/list"
	"errors"
	"io"
	"reflect"

	"github.com/russolsen/transit"
)

var (
	messageType          = reflect.TypeOf(Message{})
	pairMessageType      = reflect.TypeOf(PairMessage{})
	membershipUpdateType = reflect.TypeOf(MembershipUpdateMessage{})

	defaultMessageValueEncoder = &messageValueEncoder{}
)

// NewMessageEncoder returns a new Transit encoder
// that can encode Message values.
// More about Transit: https://github.com/cognitect/transit-format
func NewMessageEncoder(w io.Writer) *transit.Encoder {
	encoder := transit.NewEncoder(w, false)
	encoder.AddHandler(messageType, defaultMessageValueEncoder)
	encoder.AddHandler(pairMessageType, defaultMessageValueEncoder)
	encoder.AddHandler(membershipUpdateType, defaultMessageValueEncoder)
	return encoder
}

type messageValueEncoder struct{}

func (messageValueEncoder) IsStringable(reflect.Value) bool {
	return false
}

func (messageValueEncoder) Encode(e transit.Encoder, value reflect.Value, asString bool) error {
	switch message := value.Interface().(type) {
	case Message:
		taggedValue := encodeMessageToTaggedValue(message)
		return e.EncodeInterface(taggedValue, false)
	case PairMessage:
		taggedValue := transit.TaggedValue{
			Tag: pairMessageTag,
			Value: []interface{}{
				message.InstallationID,
				message.DeviceType,
				message.Name,
				message.FCMToken,
			},
		}
		return e.EncodeInterface(taggedValue, false)
	case MembershipUpdateMessage:
		updatesList := list.New()
		for _, update := range message.Updates {
			var events []interface{}
			for _, event := range update.Events {
				eventMap := map[interface{}]interface{}{
					transit.Keyword("type"):        event.Type,
					transit.Keyword("clock-value"): event.ClockValue,
				}
				if event.Name != "" {
					eventMap[transit.Keyword("name")] = event.Name
				}
				if event.Member != "" {
					eventMap[transit.Keyword("member")] = event.Member
				}
				if len(event.Members) > 0 {
					members := make([]interface{}, len(event.Members))
					for idx, m := range event.Members {
						members[idx] = m
					}
					eventMap[transit.Keyword("members")] = transit.NewSet(members)
				}
				events = append(events, eventMap)
			}

			element := map[interface{}]interface{}{
				transit.Keyword("chat-id"):   update.ChatID,
				transit.Keyword("events"):    events,
				transit.Keyword("signature"): update.Signature,
			}
			updatesList.PushBack(element)
		}
		value := []interface{}{
			message.ChatID,
			updatesList,
		}
		if message.Message != nil {
			value = append(value, encodeMessageToTaggedValue(*message.Message))
		}
		taggedValue := transit.TaggedValue{
			Tag:   membershipUpdateTag,
			Value: value,
		}
		return e.EncodeInterface(taggedValue, false)
	}

	return errors.New("unknown message type to encode")
}

func encodeMessageToTaggedValue(m Message) transit.TaggedValue {
	return transit.TaggedValue{
		Tag: messageTag,
		Value: []interface{}{
			m.Text,
			m.ContentT,
			transit.Keyword(m.MessageT),
			m.Clock,
			m.Timestamp,
			map[interface{}]interface{}{
				transit.Keyword("chat-id"): m.Content.ChatID,
				transit.Keyword("text"):    m.Content.Text,
			},
		},
	}
}
