package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type RequestPermissionsCommand struct {
}

type Permission struct {
	ParentCapability string `json:"parentCapability"`
	Date             string `json:"date"`
}

type PermissionsResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	ID      int          `json:"id"`
	Result  []Permission `json:"result"`
}

var (
	ErrNoRequestPermissionsParamsFound = errors.New("no request permission params found")
	ErrMultipleKeysFound               = errors.New("Multiple methodNames found in request permissions params")
	ErrInvalidParamType                = errors.New("Invalid parameter type")
)

func (r *RPCRequest) getRequestPermissionsParam() (string, error) {
	if r.Params == nil || len(r.Params) == 0 {
		return "", ErrEmptyRPCParams
	}

	paramMap, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return "", ErrInvalidParamType
	}

	if len(paramMap) > 1 {
		return "", ErrMultipleKeysFound
	}

	for methodName := range paramMap {
		return methodName, nil
	}

	return "", ErrNoRequestPermissionsParamsFound
}

func (c *RequestPermissionsCommand) getPermissionResponse(methodName string) (string, error) {
	date := time.Now().UnixNano() / int64(time.Millisecond)

	response := Permission{
		ParentCapability: methodName,
		Date:             fmt.Sprintf("%d", date),
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %v", err)
	}

	return string(responseJSON), nil
}

func (c *RequestPermissionsCommand) Execute(request RPCRequest) (string, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	methodName, err := request.getRequestPermissionsParam()
	if err != nil {
		return "", err
	}

	return c.getPermissionResponse(methodName)
}
