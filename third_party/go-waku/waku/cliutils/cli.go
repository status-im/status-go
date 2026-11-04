package cliutils

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type AddressValue struct {
	Value *common.Address
}

func (v *AddressValue) Set(value string) error {
	if !common.IsHexAddress(value) {
		return errors.New("invalid ethereum address")
	}

	*v.Value = common.HexToAddress(value)
	return nil
}

func (v *AddressValue) String() string {
	if v.Value == nil {
		return ""
	}
	return (*v.Value).Hex()
}

type PrivateKeyValue struct {
	Value **ecdsa.PrivateKey
}

func (v *PrivateKeyValue) Set(value string) error {
	prvKey, err := crypto.ToECDSA(common.FromHex(value))
	if err != nil {
		return errors.New("invalid private key")
	}

	*v.Value = prvKey

	return nil
}

func (v *PrivateKeyValue) String() string {
	if v.Value == nil || *v.Value == nil {
		return ""
	}
	return "0x" + common.Bytes2Hex(crypto.FromECDSA(*v.Value))
}

type ChoiceValue struct {
	Choices []string // the choices that this value can take
	Value   *string  // the actual value
}

func (v *ChoiceValue) Set(value string) error {
	for _, choice := range v.Choices {
		if strings.Compare(choice, value) == 0 {
			*v.Value = value
			return nil
		}
	}
	return fmt.Errorf("%s is not a valid option. need %+v", value, v.Choices)
}

func (v *ChoiceValue) String() string {
	if v.Value == nil {
		return ""
	}
	return *v.Value
}

// OptionalUint represents a urfave/cli flag to store uint values that can be
// optionally set and not have any default value assigned to it
type OptionalUint struct {
	Value **uint
}

// Set assigns a value to the flag only if it represents a valid uint value
func (v *OptionalUint) Set(value string) error {
	if value != "" {
		uintVal, err := strconv.ParseUint(value, 10, 0)
		if err != nil {
			return err
		}
		uVal := uint(uintVal)
		*v.Value = &uVal
	} else {
		v.Value = nil
	}

	return nil
}

// String returns the string representation of the OptionalUint flag, if set
func (v *OptionalUint) String() string {
	if v.Value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *v.Value)
}
