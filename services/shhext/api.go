package shhext

import (
	"github.com/ethereum/go-ethereum/log"
)

// PublicAPI extends whisper public API.
type PublicAPI struct {
	service *Service
	log     log.Logger
}

// NewPublicAPI returns instance of the public API.
func NewPublicAPI(s *Service) *PublicAPI {
	return &PublicAPI{
		service: s,
		log:     log.New("package", "status-go/services/sshext.PublicAPI"),
	}
}
