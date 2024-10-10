package requests

type EncodeFunctionCall struct {
	Method     string `json:"method"`
	ParamsJSON string `json:"paramsJSON"`
}
