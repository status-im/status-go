package waku

import (
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/rlp"
)

// statusOptions defines additional information shared between peers
// during the handshake.
// There might be more options provided then fields in statusOptions
// and they should be ignored during deserialization to stay forward compatible.
// In the case of RLP, options should be serialized to an array of tuples
// where the first item is a field name and the second is a RLP-serialized value.
type statusOptions struct {
	PoWRequirement       uint64      `rlp:"key=0"` // RLP does not support float64 natively
	BloomFilter          []byte      `rlp:"key=1"`
	LightNodeEnabled     bool        `rlp:"key=2"`
	ConfirmationsEnabled bool        `rlp:"key=3"`
	RateLimits           RateLimits  `rlp:"key=4"`
	TopicInterest        []TopicType `rlp:"key=5"`
}

var idxFieldKey = make(map[int]string)
var keyFieldIdx = func() map[string]int {
	result := make(map[string]int)
	opts := statusOptions{}
	v := reflect.ValueOf(opts)
	for i := 0; i < v.NumField(); i++ {
		// skip unexported fields
		if !v.Field(i).CanInterface() {
			continue
		}
		rlpTag := v.Type().Field(i).Tag.Get("rlp")
		// skip fields without rlp field tag
		if rlpTag == "" {
			continue
		}
		key := strings.Split(rlpTag, "=")[1]
		result[key] = i
		idxFieldKey[i] = key
	}
	return result
}()

func (o statusOptions) PoWRequirementF() float64 {
	return math.Float64frombits(o.PoWRequirement)
}

func (o *statusOptions) SetPoWRequirementFromF(val float64) {
	o.PoWRequirement = math.Float64bits(val)
}

func (o statusOptions) EncodeRLP(w io.Writer) error {
	v := reflect.ValueOf(o)
	optionsList := make([]interface{}, 0, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		value := v.Field(i).Interface()
		key, ok := idxFieldKey[i]
		if !ok {
			continue
		}
		optionsList = append(optionsList, []interface{}{key, value})
	}
	return rlp.Encode(w, optionsList)
}

func (o *statusOptions) DecodeRLP(s *rlp.Stream) error {
	_, err := s.List()
	if err != nil {
		return fmt.Errorf("expected an outer list: %w", err)
	}

	v := reflect.ValueOf(o)

loop:
	for {
		_, err := s.List()
		switch err {
		case nil:
			// continue to decode a key
		case rlp.EOL:
			break loop
		default:
			return fmt.Errorf("expected an inner list: %w", err)
		}
		var key string
		if err := s.Decode(&key); err != nil {
			return fmt.Errorf("invalid key: %w", err)
		}
		// Skip processing if a key does not exist.
		// It might happen when there is a new peer
		// which supports a new option with
		// a higher index.
		idx, ok := keyFieldIdx[key]
		if !ok {
			// Read the rest of the list items and dump them.
			_, err := s.Raw()
			if err != nil {
				return fmt.Errorf("failed to read the value of key %s: %w", key, err)
			}
			continue
		}
		if err := s.Decode(v.Elem().Field(idx).Addr().Interface()); err != nil {
			return fmt.Errorf("failed to decode an option %s: %w", key, err)
		}
		if err := s.ListEnd(); err != nil {
			return err
		}
	}

	return s.ListEnd()
}

func (o statusOptions) Validate() error {
	if len(o.TopicInterest) > 1000 {
		return errors.New("topic interest is limited by 1000 items")
	}
	return nil
}
