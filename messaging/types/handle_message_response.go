package types

import "crypto/ecdsa"

type HandleMessageResponse struct {
	Hash             []byte
	StatusMessages   []*Message
	DatasyncSender   *ecdsa.PublicKey
	DatasyncAcks     [][]byte
	DatasyncOffers   []DatasyncOffer
	DatasyncRequests [][]byte
}

type DatasyncOffer struct {
	GroupID   []byte
	MessageID []byte
}
