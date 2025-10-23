package types

type HandleMessageResponse struct {
	StatusMessages []*Message
	DatasyncAcks   [][]byte
}
