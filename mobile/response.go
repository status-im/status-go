package statusgo

import (
	"encoding/json"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/keystore"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

const (
	codeUnknown int = iota
	// special codes
	codeFailedParseResponse
	codeFailedParseParams
	// account related codes
	codeErrNoAccountSelected
	codeErrInvalidTxSender
	codeErrDecrypt
)

var errToCodeMap = map[error]int{
	account.ErrNoAccountSelected:   codeErrNoAccountSelected,
	wallettypes.ErrInvalidTxSender: codeErrInvalidTxSender,
	keystore.ErrDecrypt:            codeErrDecrypt,
}

type jsonrpcSuccessfulResponse struct {
	Result interface{} `json:"result"`
}

type jsonrpcErrorResponse struct {
	Error jsonError `json:"error"`
}

type jsonError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message"`
}

func prepareJSONResponse(result interface{}, err error) string {
	code := codeUnknown
	if c, ok := errToCodeMap[err]; ok {
		code = c
	}

	return prepareJSONResponseWithCode(result, err, code)
}

func prepareJSONResponseWithCode(result interface{}, err error, code int) string {
	if err != nil {
		errResponse := jsonrpcErrorResponse{
			Error: jsonError{Code: code, Message: err.Error()},
		}
		response, _ := json.Marshal(&errResponse)
		return string(response)
	}

	data, err := json.Marshal(jsonrpcSuccessfulResponse{result})
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseResponse)
	}
	return string(data)
}
