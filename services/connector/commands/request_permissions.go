package commands

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type RequestPermissionsCommand struct{}

type Permission struct {
	ParentCapability string `json:"parentCapability"`
	Date             string `json:"date"`
}

var (
	ErrNoRequestPermissionsParamsFound = errors.New("no request permission params found")
	ErrMultipleKeysFound               = errors.New("multiple methodNames found in request permissions params")
	ErrInvalidParamType                = errors.New("invalid parameter type")
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

func (c *RequestPermissionsCommand) getPermissionResponse(methodName string) Permission {
	date := time.Now().UnixNano() / int64(time.Millisecond)

	response := Permission{
		ParentCapability: methodName,
		Date:             fmt.Sprintf("%d", date),
	}

	return response
}

func (c *RequestPermissionsCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	methodName, err := request.getRequestPermissionsParam()
	if err != nil {
		return "", err
	}

	return c.getPermissionResponse(methodName), nil
}
