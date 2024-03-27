package chat

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol"
)

var (
	ErrChatNotFound            = errors.New("can't find chat")
	ErrCommunitiesNotSupported = errors.New("communities are not supported")
	ErrChatTypeNotSupported    = errors.New("chat type not supported")
)

func NewAPI(service *Service) *API {
	return &API{
		s:   service,
		log: log.New("package", "status-go/services/chat.API"),
	}
}

type API struct {
	s   *Service
	log log.Logger
}

func (api *API) EditChat(ctx context.Context, communityID types.HexBytes, chatID string, name string, color string, image images.CroppedImage) (*protocol.MessengerResponse, error) {
	if len(communityID) != 0 {
		return nil, ErrCommunitiesNotSupported
	}

	chatToEdit := api.s.messenger.Chat(chatID)
	if chatToEdit == nil {
		return nil, ErrChatNotFound
	}

	if chatToEdit.ChatType != protocol.ChatTypePrivateGroupChat {
		return nil, ErrChatTypeNotSupported
	}

	response, err := api.s.messenger.EditGroupChat(ctx, chatID, name, color, image)
	if err != nil {
		return nil, err
	}

	return response, nil
}
