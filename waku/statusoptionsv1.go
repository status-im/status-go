package waku

import (
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/rlp"
)

// statusOptions defines additional information shared between peers
// during the handshake.
// There might be more options provided then fields in statusOptions
// and they should be ignored during deserialization to stay forward compatible.
// In the case of RLP, options should be serialized to an array of tuples
// where the first item is a field name and the second is a RLP-serialized value.
type statusOptionsV1 struct {
	PoWRequirement       *uint64     `rlp:"key=0"` // RLP does not support float64 natively
	BloomFilter          []byte      `rlp:"key=1"`
	LightNodeEnabled     *bool       `rlp:"key=2"`
	ConfirmationsEnabled *bool       `rlp:"key=3"`
	RateLimits           *RateLimits `rlp:"key=4"`
	TopicInterest        []TopicType `rlp:"key=5"`
}

func (o statusOptionsV1) ToStatusOptions() statusOptions {
	return statusOptions{
		PoWRequirement:       o.PoWRequirement,
		BloomFilter:          o.BloomFilter,
		LightNodeEnabled:     o.LightNodeEnabled,
		ConfirmationsEnabled: o.ConfirmationsEnabled,
		RateLimits:           o.RateLimits,
		TopicInterest:        o.TopicInterest,
	}
}

var idxFieldKeyV1 = make(map[int]uint64)
var keyFieldIdxV1 = func() map[uint64]int {
	result := make(map[uint64]int)
	opts := statusOptionsV1{}
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
		keyString := strings.Split(rlpTag, "=")[1]
		key, err := strconv.ParseUint(keyString, 10, 64)
		if err != nil {
			panic("cannot parse uint in rlp annotation")
		}

		result[key] = i
		idxFieldKeyV1[i] = key
	}
	return result
}()

func (o statusOptionsV1) PoWRequirementF() *float64 {
	if o.PoWRequirement == nil {
		return nil
	}
	result := math.Float64frombits(*o.PoWRequirement)
	return &result
}

func (o *statusOptionsV1) SetPoWRequirementFromF(val float64) {
	requirement := math.Float64bits(val)
	o.PoWRequirement = &requirement
}

func (o statusOptionsV1) EncodeRLP(w io.Writer) error {
	v := reflect.ValueOf(o)
	var optionsList []interface{}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.IsNil() {
			value := field.Interface()
			key, ok := idxFieldKeyV1[i]
			if !ok {
				continue
			}
			if value != nil {
				optionsList = append(optionsList, []interface{}{key, value})
			}
		}
	}
	return rlp.Encode(w, optionsList)
}

func (o *statusOptionsV1) DecodeRLP(s *rlp.Stream) error {
	_, err := s.List()
	if err != nil {
		return fmt.Errorf("expected an outer list: %v", err)
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
			return fmt.Errorf("expected an inner list: %v", err)
		}
		var key uint64
		if err := s.Decode(&key); err != nil {
			return fmt.Errorf("invalid key: %v", err)
		}
		// Skip processing if a key does not exist.
		// It might happen when there is a new peer
		// which supports a new option with
		// a higher index.
		idx, ok := keyFieldIdxV1[key]
		if !ok {
			// Read the rest of the list items and dump them.
			_, err := s.Raw()
			if err != nil {
				return fmt.Errorf("failed to read the value of key %d: %v", key, err)
			}
			continue
		}
		if err := s.Decode(v.Elem().Field(idx).Addr().Interface()); err != nil {
			return fmt.Errorf("failed to decode an option %d: %v", key, err)
		}
		if err := s.ListEnd(); err != nil {
			return err
		}
	}

	return s.ListEnd()
}

func (o statusOptionsV1) Validate() error {
	if len(o.TopicInterest) > 10000 {
		return errors.New("topic interest is limited by 10000 items")
	}
	return nil
}
