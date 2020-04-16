package waku

import (
	"errors"
	"io"
	"math"

	"github.com/ethereum/go-ethereum/rlp"
)

var defaultMinPoW = math.Float64bits(0.001)

// statusOptions defines additional information shared between peers
// during the handshake.
// There might be more options provided then fields in statusOptions
// and they should be ignored during deserialization to stay forward compatible.
// In the case of RLP, options should be serialized to an array of tuples
// where the first item is a field name and the second is a RLP-serialized value.
type statusOptions struct {
	PoWRequirement       *uint64
	BloomFilter          []byte
	LightNodeEnabled     *bool
	ConfirmationsEnabled *bool
	RateLimits           *RateLimits
	TopicInterest        []TopicType
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

func (o statusOptions) EncodeRLP(w io.Writer) error {
	return errors.New("not implemented")
}

func (o *statusOptions) DecodeRLP(s *rlp.Stream) error {
	return errors.New("not implemented")
}

func (o statusOptions) PoWRequirementF() *float64 {
	if o.PoWRequirement == nil {
		return nil
	}
	result := math.Float64frombits(*o.PoWRequirement)
	return &result
}

func (o statusOptions) Validate() error {
	if len(o.TopicInterest) > 10000 {
		return errors.New("topic interest is limited by 10000 items")
	}
	return nil
}
