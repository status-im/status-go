package protocol

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
	decoder.AddHandler(pairMessageTag, pairMessageHandler)
	return decoder
}

const (
	messageTag     = "c4"
	pairMessageTag = "p2"
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
