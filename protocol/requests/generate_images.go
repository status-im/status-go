package requests

type GenerateImages struct {
	Filepath string `json:"filepath"`
	AX       int    `json:"aX"`
	AY       int    `json:"aY"`
	BX       int    `json:"bX"`
	BY       int    `json:"bY"`
}
