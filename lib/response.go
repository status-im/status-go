package main

import (
	"encoding/json"
)

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

func prepareJSONResponse(result interface{}, err error) []byte {
	if err != nil {
		errResponse := jsonrpcErrorResponse{
			Error: jsonError{Message: err.Error()},
		}
		response, _ := json.Marshal(&errResponse)
		return response
	}

	data, err := json.Marshal(jsonrpcSuccessfulResponse{result})
	if err != nil {
		return prepareJSONResponse(nil, err)
	}
	return data
}
