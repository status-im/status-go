package sdk

import (
	"encoding/json"
	"errors"
)

type shhRequest struct {
	ID      int           `json:"id"`
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type generateSymKeyFromPasswordResponse struct {
	Key string `json:"result"`
	ID  int    `json:"id"`
}

func shhGenerateSymKeyFromPasswordRequest(sdk *SDK, params []string) (res *generateSymKeyFromPasswordResponse, err error) {
	// `{"jsonrpc":"2.0","id":2950,"method":"shh_generateSymKeyFromPassword","params":["%s"]}`
	req := shhRequest{
		ID:      2950,
		JSONRPC: "2.0",
		Method:  "shh_generateSymKeyFromPassword",
	}
	for _, p := range params {
		req.Params = append(req.Params, p)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return
	}

	sdk.call(string(body), &res)

	return
}

type shhFilterFormatParam struct {
	AllowP2P bool     `json:"allowP2P"`
	Topics   []string `json:"topics"`
	Type     string   `json:"type"`
	SymKeyID string   `json:"symKeyID"`
}

type newMessageFilterResponse struct {
	FilterID string `json:"result"`
}

func newShhMessageFilterFormatRequest(sdk *SDK, topics []string, symKey string) (res newMessageFilterResponse, err error) {
	// `{"jsonrpc":"2.0","id":2,"method":"shh_newMessageFilter","params":[{"allowP2P":true,"topics":["%s"],"type":"sym","symKeyID":"%s"}]}`
	req := shhRequest{
		ID:      2,
		JSONRPC: "2.0",
		Method:  "shh_newMessageFilter",
	}
	req.Params = append(req.Params, &shhFilterFormatParam{
		AllowP2P: true,
		Topics:   topics,
		Type:     "sym",
		SymKeyID: symKey,
	})

	body, err := json.Marshal(req)
	if err != nil {
		return
	}

	sdk.call(string(body), &res)

	return
}

type web3ShaResponse struct {
	Result string `json:"result"`
}

func web3Sha3Request(sdk *SDK, id int, params []string) (res *web3ShaResponse, err error) {
	// `{"jsonrpc":"2.0","method":"web3_sha3","params":["%s"],"id":%d}`
	req := shhRequest{
		ID:      id,
		JSONRPC: "2.0",
		Method:  "web3_sha3",
	}
	for _, p := range params {
		req.Params = append(req.Params, p)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return
	}

	sdk.call(string(body), &res)

	return
}

type statusLoginParam struct {
	Address  string `json:"address"`
	Password string `json:"password"`
}

type loginResponse struct {
	Result struct {
		AddressKeyID string `json:"address_key_id"`
	} `json:"result"`
}

func statusLoginRequest(sdk *SDK, address, password string) (res loginResponse, err error) {
	// `{"jsonrpc":"2.0","method":"status_login","params":[{"address":"%s","password":"%s"}]}`
	req := shhRequest{
		JSONRPC: "2.0",
		Method:  "status_login",
	}

	req.Params = append(req.Params, &statusLoginParam{
		Address:  address,
		Password: password,
	})

	body, err := json.Marshal(req)
	if err != nil {
		return
	}

	sdk.call(string(body), &res)

	return
}

type statusSignupParam struct {
	Password string `json:"password"`
}

type signupResponse struct {
	Result struct {
		Address  string `json:"address"`
		Pubkey   string `json:"pubkey"`
		Mnemonic string `json:"mnemonic"`
	} `json:"result"`
}

func statusSignupRequest(sdk *SDK, password string) (res signupResponse, err error) {
	// `{"jsonrpc":"2.0","method":"status_signup","params":[{"password":"%s"}]}`
	req := shhRequest{
		JSONRPC: "2.0",
		Method:  "status_signup",
	}

	req.Params = append(req.Params, &statusSignupParam{
		Password: password,
	})

	body, err := json.Marshal(req)
	if err != nil {
		return
	}

	sdk.call(string(body), &res)

	return
}

type getFilterMessagesResponse struct {
	Result interface{} `json:"result"`
}

func shhGetFilterMessagesRequest(sdk *SDK, filters []string) (res *getFilterMessagesResponse, err error) {
	// `{"jsonrpc":"2.0","id":2968,"method":"shh_getFilterMessages","params":["%s"]}`
	req := shhRequest{
		ID:      2968,
		JSONRPC: "2.0",
		Method:  "shh_getFilterMessages",
	}
	for _, f := range filters {
		req.Params = append(req.Params, f)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return
	}

	sdk.call(string(body), &res)

	return
}

type shhPostParam struct {
	Signature string  `json:"sig"`
	SymKeyID  string  `json:"symKeyID"`
	Payload   string  `json:"payload"`
	Topic     string  `json:"topic"`
	TTL       int     `json:"ttl"`
	PowTarget float64 `json:"powTarget"`
	PowTime   int     `json:"powTime"`
}

// error response {"jsonrpc":"2.0","id":633,"error":{"code":-32000,"message":"message rejected, PoW too low"}}
type sshPostError struct {
	Code    float64 `json:"code"`
	Message string  `json:"message"`
}

type shhPostResponse struct {
	Error *sshPostError `json:"error"`
}

func shhPostRequest(sdk *SDK, params []*shhPostParam) (res *shhPostResponse, err error) {
	// `{"jsonrpc":"2.0","id":633,"method":"shh_post","params":[{"sig":"%s","symKeyID":"%s","payload":"%s","topic":"%s","ttl":10,"powTarget":%g,"powTime":1}]}`
	req := shhRequest{
		ID:      633,
		JSONRPC: "2.0",
		Method:  "shh_post",
	}
	for _, p := range params {
		req.Params = append(req.Params, p)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return
	}

	err = sdk.call(string(body), &res)
	if err != nil {
		return res, err
	}

	if res.Error != nil {
		return res, errors.New(res.Error.Message)
	}

	return
}
