package bigint

import (
	"database/sql/driver"
	"errors"
	"math/big"
)

// SQLBigIntBytes type for storing big.Int as BLOB in the databse.
type SQLBigIntBytes big.Int

func (i *SQLBigIntBytes) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	val, ok := value.([]byte)
	if !ok {
		return errors.New("not an integer")
	}
	(*big.Int)(i).SetBytes(val)
	return nil
}

func (i *SQLBigIntBytes) Value() (driver.Value, error) {
	val := (*big.Int)(i)
	if val == nil {
		return nil, nil
	}
	return (*big.Int)(i).Bytes(), nil
}
