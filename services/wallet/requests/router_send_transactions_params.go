package requests

import "github.com/status-im/status-go/errors"

var (
	ErrInvalidSignatureDetails = &errors.ErrorResponse{Code: errors.ErrorCode("WT-004"), Details: "invalid signature details"}
)

type RouterSendTransactionsParams struct {
	Uuid       string                      `json:"uuid"`
	Signatures map[string]SignatureDetails `json:"signatures"`
}

type SignatureDetails struct {
	R string `json:"r"`
	S string `json:"s"`
	V string `json:"v"`
}

func (sd *SignatureDetails) Validate() error {
	if len(sd.R) != 64 || len(sd.S) != 64 || len(sd.V) != 2 {
		return ErrInvalidSignatureDetails
	}

	return nil
}
