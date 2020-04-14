package waku

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/rlp"
)

// statusOptionKey is a current type used in statusOptions as a key.
type statusOptionKey uint

// statusOptionKeyType is a type of a statusOptions key used for a particular instance of statusOptions struct.
type statusOptionKeyType uint

const (
	sOKTS statusOptionKeyType = iota + 1 // Status Option Key Type String
	sOKTU                                // Status Option Key Type Uint
)

var (
	defaultMinPoW = math.Float64bits(0.001)
	idxFieldKey   = make(map[int]statusOptionKey)
	keyFieldIdx   = make(map[statusOptionKey]int)
)

// statusOptions defines additional information shared between peers
// during the handshake.
// There might be more options provided then fields in statusOptions
// and they should be ignored during deserialization to stay forward compatible.
// In the case of RLP, options should be serialized to an array of tuples
// where the first item is a field name and the second is a RLP-serialized value.
type statusOptions struct {
	PoWRequirement       *uint64     `rlp:"key=0"` // RLP does not support float64 natively
	BloomFilter          []byte      `rlp:"key=1"`
	LightNodeEnabled     *bool       `rlp:"key=2"`
	ConfirmationsEnabled *bool       `rlp:"key=3"`
	RateLimits           *RateLimits `rlp:"key=4"`
	TopicInterest        []TopicType `rlp:"key=5"`
	keyType              statusOptionKeyType
}

// initFLPKeyFields initialises the values of `idxFieldKey` and `keyFieldIdx`
func initRLPKeyFields() error {
	o := statusOptions{}
	v := reflect.ValueOf(o)

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

		keys := strings.Split(rlpTag, "=")

		if len(keys) != 2 || keys[0] != "key" {
			panic("invalid value of \"rlp\" tag, expected \"key=N\" where N is uint")
		}

		// parse keys[1] as an uint
		key, err := strconv.ParseUint(keys[1], 10, 64)
		if err != nil {
			return fmt.Errorf("malformed rlp tag '%s', expected \"key=N\" where N is uint: %v", rlpTag, err)
		}

		// typecast key to be of statusOptionKey type
		keyFieldIdx[statusOptionKey(key)] = i
		idxFieldKey[i] = statusOptionKey(key)
	}

	return nil
}

// WithDefaults adds the default values for a given peer.
// This are not the host default values, but the default values that ought to
// be used when receiving from an update from a peer.
func (o statusOptions) WithDefaults() statusOptions {
	if o.PoWRequirement == nil {
		o.PoWRequirement = &defaultMinPoW
	}

	if o.LightNodeEnabled == nil {
		lightNodeEnabled := false
		o.LightNodeEnabled = &lightNodeEnabled
	}

	if o.ConfirmationsEnabled == nil {
		confirmationsEnabled := false
		o.ConfirmationsEnabled = &confirmationsEnabled
	}

	if o.RateLimits == nil {
		o.RateLimits = &RateLimits{}
	}

	if o.BloomFilter == nil {
		o.BloomFilter = MakeFullNodeBloom()
	}

	return o
}

func (o statusOptions) PoWRequirementF() *float64 {
	if o.PoWRequirement == nil {
		return nil
	}
	result := math.Float64frombits(*o.PoWRequirement)
	return &result
}

func (o *statusOptions) SetPoWRequirementFromF(val float64) {
	requirement := math.Float64bits(val)
	o.PoWRequirement = &requirement
}

func (o statusOptions) EncodeRLP(w io.Writer) error {
	v := reflect.ValueOf(o)
	var optionsList []interface{}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		// skip unexported fields
		if !field.CanInterface() {
			continue
		}

		if field.IsNil() {
			continue
		}

		value := field.Interface()
		key, ok := idxFieldKey[i]
		if !ok {
			continue
		}

		if value != nil {
			optionsList = append(optionsList, []interface{}{o.encodeKey(key), value})
		}
	}
	return rlp.Encode(w, optionsList)
}

func (o *statusOptions) DecodeRLP(s *rlp.Stream) error {
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

		key, keyType, err := o.decodeKey(s)
		if err != nil {
			return fmt.Errorf("key decode failure: %v", err)
		}
		o.setKeyType(keyType)

		// Skip processing if a key does not exist.
		// It might happen when there is a new peer
		// which supports a new option with
		// a higher index.
		idx, ok := keyFieldIdx[key]
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

func (o statusOptions) decodeKey(s *rlp.Stream) (statusOptionKey, statusOptionKeyType, error) {
	// Problem: A string will be encoded to bytes, and bytes can be decoded into a uint.
	// This means that an encoded string that is attempted to be decoded into a uint will succeed and return a valid uint.
	// This is bad because wildly inaccurate keys can be returned. See below examples:
	// - string("0"); encodes to byte(48); decodes to uint(48).
	// - string("111"); encodes to []byte(131, 49, 49, 49); decode to uint(3223857).
	// This means an expected index of 0 will be returned as 48. An expected index of 111 will be returned as 3223857

	// Solution: We need to first test if the RLP stream can be decoded into a string.
	// If a stream can be decoded into a string, attempt to decode the string into a uint.
	// If decoding the string into a uint is successful return the value.
	// If decoding the string failed, attempt to decode as a uint. Return the result or error from this final step.

	// decode into bytes, detect if bytes can be parsed as a string and from a string to a uint
	var bKey []byte
	if err := s.Decode(&bKey); err != nil {
		return 0, 0, err
	}

	// Parse string into uint
	uKey, err := strconv.ParseUint(string(bKey), 10, 64)
	if err == nil {
		return statusOptionKey(uKey), sOKTS, err
	}

	// If statusOptionKey (uint) can be decoded return it
	buf := bytes.NewBuffer(bKey)
	uintKey, c := binary.ReadUvarint(buf)
	spew.Dump(uintKey, c)

	return statusOptionKey(uintKey), sOKTU, nil
}

// setKeyType sets a statusOptions' keyType if it hasn't previously been set
func (o *statusOptions) setKeyType(t statusOptionKeyType) {
	if o.keyType == 0 {
		o.keyType = t
	}
}

func (o statusOptions) encodeKey(key statusOptionKey) interface{} {
	if o.keyType == sOKTS {
		return fmt.Sprint(key)
	}

	return key
}

func (o statusOptions) Validate() error {
	if len(o.TopicInterest) > 1000 {
		return errors.New("topic interest is limited by 1000 items")
	}
	return nil
}
