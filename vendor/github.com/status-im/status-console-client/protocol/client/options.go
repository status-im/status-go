package client

import (
	"fmt"

	"github.com/status-im/status-console-client/protocol/v1"
)

var (
	errUnsupportedContactType = fmt.Errorf("unsupported contact type")
)

func createSubscribeOptions(c Contact) (opts protocol.SubscribeOptions, err error) {
	opts.ChatName = c.Topic
	switch c.Type {
	case ContactPublicRoom:
	case ContactPrivate:
		opts.Recipient = c.PublicKey
	default:
		err = errUnsupportedContactType
	}
	return
}

func createSendOptions(c Contact) (opts protocol.SendOptions, err error) {
	opts.ChatName = c.Topic
	switch c.Type {
	case ContactPublicRoom:
	case ContactPrivate:
		opts.Recipient = c.PublicKey
	default:
		err = errUnsupportedContactType
	}
	return
}

func enhanceRequestOptions(c Contact, opts *protocol.RequestOptions) error {
	var chatOptions protocol.ChatOptions
	chatOptions.ChatName = c.Topic
	switch c.Type {
	case ContactPublicRoom:
	case ContactPrivate:
		chatOptions.Recipient = c.PublicKey
	default:
		return errUnsupportedContactType
	}

	opts.Chats = append(opts.Chats, chatOptions)

	return nil
}
