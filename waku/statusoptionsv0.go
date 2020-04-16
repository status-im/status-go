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

// DEPRECATED: This is now deprecated. It incorrectly uses ascii keys for
// RLP serialization

type statusOptionsV0 struct {
	PoWRequirement       *uint64     `rlp:"key=0"` // RLP does not support float64 natively
	BloomFilter          []byte      `rlp:"key=1"`
	LightNodeEnabled     *bool       `rlp:"key=2"`
	ConfirmationsEnabled *bool       `rlp:"key=3"`
	RateLimits           *RateLimits `rlp:"key=4"`
	TopicInterest        []TopicType `rlp:"key=5"`
}

func (o statusOptionsV0) ToStatusOptions() statusOptions {
	return statusOptions{
		PoWRequirement:       o.PoWRequirement,
		BloomFilter:          o.BloomFilter,
		LightNodeEnabled:     o.LightNodeEnabled,
		ConfirmationsEnabled: o.ConfirmationsEnabled,
		RateLimits:           o.RateLimits,
		TopicInterest:        o.TopicInterest,
	}
}

var idxFieldKeyV0 = make(map[int]string)
var keyFieldIdxV0 = func() map[string]int {
	result := make(map[string]int)
	opts := statusOptionsV0{}
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
		idxFieldKeyV0[i] = key
	}
	return result
}()

func (o *statusOptionsV0) SetPoWRequirementFromF(val float64) {
	requirement := math.Float64bits(val)
	o.PoWRequirement = &requirement
}

func (o statusOptionsV0) EncodeRLP(w io.Writer) error {
	v := reflect.ValueOf(o)
	var optionsList []interface{}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.IsNil() {
			value := field.Interface()
			key, ok := idxFieldKeyV0[i]
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

func (o *statusOptionsV0) DecodeRLP(s *rlp.Stream) error {
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
		var key string
		if err := s.Decode(&key); err != nil {
			return fmt.Errorf("invalid key: %v", err)
		}
		// Skip processing if a key does not exist.
		// It might happen when there is a new peer
		// which supports a new option with
		// a higher index.
		idx, ok := keyFieldIdxV0[key]
		if !ok {
			// Read the rest of the list items and dump them.
			_, err := s.Raw()
			if err != nil {
				return fmt.Errorf("failed to read the value of key %s: %v", key, err)
			}
			continue
		}
		if err := s.Decode(v.Elem().Field(idx).Addr().Interface()); err != nil {
			return fmt.Errorf("failed to decode an option %s: %v", key, err)
		}
		if err := s.ListEnd(); err != nil {
			return err
		}
	}

	return s.ListEnd()
}

func (o statusOptionsV0) Validate() error {
	if len(o.TopicInterest) > 10000 {
		return errors.New("topic interest is limited by 10000 items")
	}
	return nil
}
