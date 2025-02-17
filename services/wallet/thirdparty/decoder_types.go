package thirdparty

//go:generate mockgen -package=mock_thirdparty -source=decoder_types.go -destination=mock/decoder_types.go

type DataParsed struct {
	Name      string            `json:"name"`
	ID        string            `json:"id"`
	Inputs    map[string]string `json:"inputs"`
	Signature string            `json:"signature"`
}

type DecoderProvider interface {
	Run(data string) (*DataParsed, error)
}
