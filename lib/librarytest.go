// +build e2e_test

// This is a file with e2e tests for C bindings written in library.go.
// As a CGO file, it can't have `_test.go` suffix as it's not allowed by Go.
// At the same time, we don't want this file to be included in the binaries.
// This is why `e2e_test` tag was introduced. Without it, this file is excluded
// from the build. Providing this tag will include this file into the build
// and that's what is done while running e2e tests for `lib/` package.

// Additionaly this file should contain test that mock the Status API.
// Existing test in 'utils.go' that test the Status API will be migrated to the
// e2e package and test that test the C Binding will be migrated to this file

package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/geth/common"
	"github.com/stretchr/testify/assert"
)

func testCreateAccountWithMock(t *testing.T) {
	realStatusAPI := statusAPI
	defer func() { statusAPI = realStatusAPI }()

	// Setup Mock StatusAPI
	ctrl := gomock.NewController(t)
	status := NewMockStatusAPI(ctrl)
	statusAPI = status
	accountInfo1 := common.AccountInfo{Address: "add", Mnemonic: "mne", PubKey: "Pub"}
	accountInfo2 := common.AccountInfo{Error: "Error Message"}
	status.EXPECT().CreateAccount("pass1").Return(accountInfo1)
	status.EXPECT().CreateAccount("").Return(accountInfo1)
	status.EXPECT().CreateAccount(C.GoString(nil)).Return(accountInfo1)
	status.EXPECT().CreateAccount("pass2").Return(accountInfo2)

	// C Strings
	pass1 := C.CString("pass1")
	pass2 := C.CString("pass2")
	empty := C.CString("")
	jsonResult := C.CString(`{"address":"add","pubkey":"Pub","mnemonic":"mne","error":""}`)
	jsonResultError := C.CString(`{"address":"","pubkey":"","mnemonic":"","error":"Error Message"}`)
	defer func() {
		C.free(unsafe.Pointer(pass1))
		C.free(unsafe.Pointer(pass2))
		C.free(unsafe.Pointer(empty))
		C.free(unsafe.Pointer(jsonResult))
		C.free(unsafe.Pointer(jsonResultError))
	}()

	tests := []struct {
		name     string
		password *C.char
		want     *C.char
	}{
		{"testCreateAccountWithMock/Normal", pass1, jsonResult},
		{"testCreateAccountWithMock/EmptyParam", empty, jsonResult},
		{"testCreateAccountWithMock/NilParam", nil, jsonResult},
		{"testCreateAccountWithMock/ErrorResult", pass2, jsonResultError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateAccount(tt.password); C.GoString(got) != C.GoString(tt.want) {
				assert.Equal(t, C.GoString(tt.want), C.GoString(got))
			}
		})
	}
}

func testCreateChildAccountWithMock(t *testing.T) {
	realStatusAPI := statusAPI
	defer func() { statusAPI = realStatusAPI }()

	// Setup Mock StatusAPI
	ctrl := gomock.NewController(t)
	status := NewMockStatusAPI(ctrl)
	statusAPI = status

	accountInfo1 := common.AccountInfo{Address: "add", PubKey: "Pub"}
	accountInfo2 := common.AccountInfo{Error: "Error Message", ErrorValue: fmt.Errorf("Error Message")}
	status.EXPECT().CreateChildAccount("parent1", "pass1").Return(accountInfo1)
	status.EXPECT().CreateChildAccount("", "").Return(accountInfo1).AnyTimes()
	status.EXPECT().CreateChildAccount("parent2", "pass2").Return(accountInfo2)

	// C Strings
	pass1 := C.CString("pass1")
	pass2 := C.CString("pass2")
	parent1 := C.CString("parent1")
	parent2 := C.CString("parent2")
	empty := C.CString("")
	jsonResult := C.CString(`{"address":"add","pubkey":"Pub","mnemonic":"","error":""}`)
	jsonResultError := C.CString(`{"address":"","pubkey":"","mnemonic":"","error":"Error Message"}`)
	defer func() {
		C.free(unsafe.Pointer(pass1))
		C.free(unsafe.Pointer(pass2))
		C.free(unsafe.Pointer(parent1))
		C.free(unsafe.Pointer(parent2))
		C.free(unsafe.Pointer(empty))
		C.free(unsafe.Pointer(jsonResult))
		C.free(unsafe.Pointer(jsonResultError))
	}()

	tests := []struct {
		name     string
		parrent  *C.char
		password *C.char
		want     *C.char
	}{
		{"testCreateChildAccountWithMock/Normal", parent1, pass1, jsonResult},
		{"testCreateChildAccountWithMock/EmptyParam", empty, empty, jsonResult},
		{"testCreateChildAccountWithMock/NilParam", nil, nil, jsonResult},
		{"testCreateChildAccountWithMock/ErrorResult", parent2, pass2, jsonResultError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateChildAccount(tt.parrent, tt.password); C.GoString(got) != C.GoString(tt.want) {
				assert.Equal(t, C.GoString(tt.want), C.GoString(got))
			}
		})
	}

}

func testRecoverAccountWithMock(t *testing.T) {
	realStatusAPI := statusAPI
	defer func() { statusAPI = realStatusAPI }()

	// Setup Mock StatusAPI
	ctrl := gomock.NewController(t)
	status := NewMockStatusAPI(ctrl)
	statusAPI = status

	accountInfo1 := common.AccountInfo{Address: "add", PubKey: "Pub", Mnemonic: "mnemonic"}
	accountInfo2 := common.AccountInfo{Error: "Error Message", ErrorValue: fmt.Errorf("Error Message")}
	status.EXPECT().RecoverAccount("pass1", "mnemonic1").Return(accountInfo1)
	status.EXPECT().RecoverAccount("", "").Return(accountInfo1).AnyTimes()
	status.EXPECT().RecoverAccount("pass2", "mnemonic2").Return(accountInfo2)

	// C Strings
	pass1 := C.CString("pass1")
	pass2 := C.CString("pass2")
	mnemonic1 := C.CString("mnemonic1")
	mnemonic2 := C.CString("mnemonic2")
	empty := C.CString("")
	jsonResult := C.CString(`{"address":"add","pubkey":"Pub","mnemonic":"mnemonic","error":""}`)
	jsonResultError := C.CString(`{"address":"","pubkey":"","mnemonic":"","error":"Error Message"}`)
	defer func() {
		C.free(unsafe.Pointer(pass1))
		C.free(unsafe.Pointer(pass2))
		C.free(unsafe.Pointer(mnemonic1))
		C.free(unsafe.Pointer(mnemonic2))
		C.free(unsafe.Pointer(empty))
		C.free(unsafe.Pointer(jsonResult))
		C.free(unsafe.Pointer(jsonResultError))
	}()

	tests := []struct {
		name     string
		password *C.char
		mnemonic *C.char
		want     *C.char
	}{
		{"testRecoverAccountWithMock/Normal", pass1, mnemonic1, jsonResult},
		{"testRecoverAccountWithMock/EmptyParam", empty, empty, jsonResult},
		{"testRecoverAccountWithMock/NilParam", nil, nil, jsonResult},
		{"testRecoverAccountWithMock/ErrorResult", pass2, mnemonic2, jsonResultError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RecoverAccount(tt.password, tt.mnemonic); C.GoString(got) != C.GoString(tt.want) {
				assert.Equal(t, C.GoString(tt.want), C.GoString(got))
			}
		})
	}

}

func testValidateNodeConfigWithMock(t *testing.T) {
	realStatusAPI := statusAPI
	defer func() { statusAPI = realStatusAPI }()

	// Setup Mock StatusAPI
	ctrl := gomock.NewController(t)
	status := NewMockStatusAPI(ctrl)
	statusAPI = status

	apiDetailedResponse1 := common.APIDetailedResponse{Status: true}
	apiDetailedResponse2 := common.APIDetailedResponse{Status: false, FieldErrors: []common.APIFieldError{
		{Parameter: "param1", Errors: []common.APIError{{Message: "perror1"}, {Message: "perror2"}}},
		{Parameter: "param2", Errors: []common.APIError{{Message: "perror1"}}},
	}}
	apiDetailedResponse3 := common.APIDetailedResponse{}

	status.EXPECT().ValidateJSONConfig("{json1}").Return(apiDetailedResponse1)
	status.EXPECT().ValidateJSONConfig("{json2}").Return(apiDetailedResponse2)
	status.EXPECT().ValidateJSONConfig("").Return(apiDetailedResponse3).AnyTimes()

	// C Strings
	//TODO should the CStrings be C.free
	config1 := C.CString("{json1}")
	config2 := C.CString("{json2}")
	empty := C.CString("")
	jsonResult1 := C.CString(`{"status":true}`)
	jsonResult2 := C.CString(`{"status":false,"field_errors":[{"parameter":"param1","errors":[{"message":"perror1"},{"message":"perror2"}]},{"parameter":"param2","errors":[{"message":"perror1"}]}]}`)
	jsonResult3 := C.CString(`{"status":false}`)
	defer func() {
		C.free(unsafe.Pointer(config1))
		C.free(unsafe.Pointer(config2))
		C.free(unsafe.Pointer(empty))
		C.free(unsafe.Pointer(jsonResult1))
		C.free(unsafe.Pointer(jsonResult2))
		C.free(unsafe.Pointer(jsonResult3))
	}()

	tests := []struct {
		name       string
		configJSON *C.char
		want       *C.char
	}{
		{"testValidateNodeConfigWithMock/Normal", config1, jsonResult1},
		{"testValidateNodeConfigWithMock/ValidationErrors", config2, jsonResult2},
		{"testValidateNodeConfigWithMock/emptyconfig", empty, jsonResult3},
		{"testValidateNodeConfigWithMock/nilconfig", nil, jsonResult3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateNodeConfig(tt.configJSON); C.GoString(got) != C.GoString(tt.want) {
				assert.Equal(t, C.GoString(tt.want), C.GoString(got))
			}
		})
	}

}
