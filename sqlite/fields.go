package sqlite

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"reflect"
)

// JSONBlob type for marshaling/unmarshaling inner type to json.
type JSONBlob struct {
	Data interface{}
}

// Scan implements interface.
func (blob *JSONBlob) Scan(value interface{}) error {
	dataVal := reflect.ValueOf(blob.Data)
	if value == nil || dataVal.Kind() == reflect.Ptr && dataVal.IsNil() {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("not a byte slice")
	}
	if len(bytes) == 0 {
		return nil
	}
	err := json.Unmarshal(bytes, blob.Data)
	return err
}

// Value implements interface.
func (blob *JSONBlob) Value() (driver.Value, error) {
	dataVal := reflect.ValueOf(blob.Data)
	if blob.Data == nil || dataVal.Kind() == reflect.Ptr && dataVal.IsNil() {
		return nil, nil
	}
	return json.Marshal(blob.Data)
}
