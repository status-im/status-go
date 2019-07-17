package statusproto

import (
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
	decoder.AddHandler(messageTag, statusMessageHandler)
	decoder.AddHandler(pairMessageTag, pairMessageHandler)
	return decoder
}

const (
	messageTag     = "c4"
	pairMessageTag = "p2"
)

func statusMessageHandler(d transit.Decoder, value interface{}) (interface{}, error) {
	taggedValue, ok := value.(transit.TaggedValue)
	if !ok {
		return nil, errors.New("not a tagged value")
	}
	values, ok := taggedValue.Value.([]interface{})
	if !ok {
		return nil, errors.New("tagged value does not contain values")
	}

	sm := Message{}
	for idx, v := range values {
		var ok bool

		switch idx {
		case 0:
			sm.Text, ok = v.(string)
		case 1:
			sm.ContentT, ok = v.(string)
		case 2:
			var messageT transit.Keyword
			messageT, ok = v.(transit.Keyword)
			if ok {
				sm.MessageT = string(messageT)
			}
		case 3:
			sm.Clock, ok = v.(int64)
		case 4:
			var timestamp int64
			timestamp, ok = v.(int64)
			if ok {
				sm.Timestamp = TimestampInMs(timestamp)
			}
		case 5:
			var content map[interface{}]interface{}
			content, ok = v.(map[interface{}]interface{})
			if !ok {
				break
			}

			for key, contentVal := range content {
				var keyKeyword transit.Keyword
				keyKeyword, ok = key.(transit.Keyword)
				if !ok {
					break
				}

				switch keyKeyword {
				case transit.Keyword("text"):
					sm.Content.Text, ok = contentVal.(string)
				case transit.Keyword("chat-id"):
					sm.Content.ChatID, ok = contentVal.(string)
				}
			}
		default:
			// skip any other values
			ok = true
		}

		if !ok {
			return nil, fmt.Errorf("invalid value for index: %d", idx)
		}
	}
	return sm, nil
}

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
