package requests

import "github.com/status-im/status-go/services/wallet/transfer"

type RouterSendTransactionsParams struct {
	Uuid       string                               `json:"uuid"`
	Signatures map[string]transfer.SignatureDetails `json:"signatures"`
}
