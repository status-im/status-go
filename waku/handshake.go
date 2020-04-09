package waku

import (
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/rlp"
)

// statusOptionKey is a current type used in statusOptions as a key.
type statusOptionKey uint

// statusOptionKeyType is a type of a statusOptions key used for a particular instance of statusOptions struct.
type statusOptionKeyType uint

type statusOptionKeyToType struct {
	Idx  int
	Key  statusOptionKey
	Type statusOptionKeyType
}

const (
	sOKTS statusOptionKeyType = iota + 1 // Status Option Key Type String
	sOKTU                            // Status Option Key Type Uint
)

var (
	defaultMinPoW = math.Float64bits(0.001)
	idxFieldKey   = map[int]statusOptionKey{
		0: 0,
		1: 1,
		2: 2,
		3: 3,
		4: 4,
		5: 5,
	}
	keyFieldIdx = map[statusOptionKey]int{
		0: 0,
		1: 1,
		2: 2,
		3: 3,
		4: 4,
		5: 5,
	}
)

type keyTypeMapping struct {
	idxFieldKey map[int]*statusOptionKeyToType
	keyFieldIdx map[statusOptionKey]*statusOptionKeyToType
}

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
	keyTypeMapping       keyTypeMapping
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

		var val []interface{}

		if o.keyTypeMapping.idxFieldKey == nil {
			val = append(val, key, value)
		}

		k, ok := o.keyTypeMapping.idxFieldKey[i]
		if !ok {
			val = append(val, key, value)
		} else {
			ke, err := k.encode()
			if err != nil {
				return fmt.Errorf("key encoding fail: %v", err)
			}
			val = append(val, ke, value)
		}

		if value != nil {
			optionsList = append(optionsList, val)
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

		ktt := statusOptionKeyToType{}
		if err = ktt.decodeStream(s); err != nil {
			return fmt.Errorf("invalid key: %v", err)
		}

		// Skip processing if a key does not exist.
		// It might happen when there is a new peer
		// which supports a new option with
		// a higher index.
		idx, ok := keyFieldIdx[ktt.Key]
		if !ok {
			// Read the rest of the list items and dump them.
			_, err := s.Raw()
			if err != nil {
				return fmt.Errorf("failed to read the value of key %d: %v", ktt.Key, err)
			}
			continue
		}

		ktt.Idx = idx
		o.addKeyToType(&ktt)

		if err := s.Decode(v.Elem().Field(idx).Addr().Interface()); err != nil {
			return fmt.Errorf("failed to decode an option %d: %v", ktt.Key, err)
		}
		if err := s.ListEnd(); err != nil {
			return err
		}
	}

	return s.ListEnd()
}

func (o *statusOptions) addKeyToType(ktt *statusOptionKeyToType) {

	if o.keyTypeMapping.idxFieldKey == nil {
		o.keyTypeMapping.idxFieldKey = make(map[int]*statusOptionKeyToType)
	}

	if o.keyTypeMapping.keyFieldIdx == nil {
		o.keyTypeMapping.keyFieldIdx = make(map[statusOptionKey]*statusOptionKeyToType)
	}

	o.keyTypeMapping.idxFieldKey[ktt.Idx] = ktt
	o.keyTypeMapping.keyFieldIdx[ktt.Key] = ktt
}

func (k *statusOptionKeyToType) decodeStream(s *rlp.Stream) error {
	var key statusOptionKey

	// If uint can be decoded return it
	if err := s.Decode(&key); err == nil {
		k.Key = key
		k.Type = sOKTU
		return nil
	}

	// Attempt decoding into a string
	var sKey string
	if err := s.Decode(&sKey); err != nil {
		return err
	}

	// Parse string into uint
	uKey, err := strconv.ParseUint(sKey, 10, 64)
	if err != nil {
		return err
	}

	k.Key = statusOptionKey(uKey)
	k.Type = sOKTS
	return nil
}

func (k statusOptionKeyToType) encode() (interface{}, error) {
	switch k.Type {
	case sOKTU:
		return k.Key, nil
	case sOKTS:
		return fmt.Sprint(k.Key), nil
	default:
		return nil, fmt.Errorf("failed to encode key '%d', unknown key type '%d'", k.Key, k.Type)
	}
}

func (o statusOptions) Validate() error {
	if len(o.TopicInterest) > 1000 {
		return errors.New("topic interest is limited by 1000 items")
	}
	return nil
}
