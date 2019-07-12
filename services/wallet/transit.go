package wallet

import (
	"errors"
	"io"
	"reflect"

	"github.com/russolsen/transit"
)

func NewEncoder(w io.Writer) *transit.Encoder {
	encoder := transit.NewEncoder(w, false)
	encoder.AddHandler(reflect.TypeOf(Browser{}), browserEncoder{})
	return encoder
}

const (
	browserTag = "b1"
)

type browserEncoder struct{}

func (browserEncoder) IsStringable(reflect.Value) bool {
	return false
}

func (browserEncoder) Encode(e transit.Encoder, value reflect.Value, asString bool) error {
	switch val := value.Interface().(type) {
	case Browser:
		return e.EncodeInterface(transit.TaggedValue{
			Tag: browserTag,
			Value: []interface{}{
				val.ID,
				val.Name,
				val.Timestamp,
				val.Dapp,
				val.HistoryIndex,
				val.History,
			}}, false)
	}
	return errors.New("unknown type")
}
