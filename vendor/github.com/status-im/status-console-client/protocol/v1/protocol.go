package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"time"
)

// Protocol is an interface defining basic methods to receive and send messages.
type Protocol interface {
	// Subscribe listens to new messages.
	Subscribe(ctx context.Context, messages chan<- *Message, options SubscribeOptions) (*Subscription, error)

	// Send sends a message to the network.
	// Identity is required as the protocol requires
	// all messages to be signed.
	Send(ctx context.Context, data []byte, options SendOptions) ([]byte, error)

	// Request retrieves historic messages.
	Request(ctx context.Context, params RequestOptions) error
}

// ChatOptions are chat specific options, usually related to the recipient/destination.
type ChatOptions struct {
	ChatName  string           // for public chats
	Recipient *ecdsa.PublicKey // for private chats
}

func (o ChatOptions) Validate() error {
	if o == (ChatOptions{}) {
		return errors.New("empty options")
	}
	return nil
}

const (
	DefaultDurationRequestOptions = 24 * time.Hour
)

// RequestOptions is a list of params required
// to request for historic messages.
type RequestOptions struct {
	Chats []ChatOptions
	Limit int
	From  int64 // in seconds
	To    int64 // in seconds
}

// DefaultRequestOptions returns default options returning messages
// from the last 24 hours.
func DefaultRequestOptions() RequestOptions {
	return RequestOptions{
		From:  time.Now().Add(-DefaultDurationRequestOptions).Unix(),
		To:    time.Now().Unix(),
		Limit: 1000,
	}
}

// FromAsTime converts int64 (timestamp in seconds) to time.Time.
func (o RequestOptions) FromAsTime() time.Time {
	return time.Unix(o.From, 0)
}

// ToAsTime converts int64 (timestamp in seconds) to time.Time.
func (o RequestOptions) ToAsTime() time.Time {
	return time.Unix(o.To, 0)
}

func (o RequestOptions) Equal(someOpts RequestOptions) bool {
	for i, chat := range o.Chats {
		if len(someOpts.Chats) < i {
			if chat != someOpts.Chats[i] {
				return false
			}
		}
	}

	return (o.From == someOpts.From &&
		o.To == someOpts.To &&
		o.Limit == someOpts.Limit)
}

// Validate verifies that the given request options are valid.
func (o RequestOptions) Validate() error {
	if len(o.Chats) == 0 {
		return errors.New("no chats selected")
	}

	for _, chatOpts := range o.Chats {
		if err := chatOpts.Validate(); err != nil {
			return err
		}
	}

	if o.To == 0 {
		return errors.New("invalid 'to' field")
	}

	if o.To <= o.From {
		return errors.New("invalid 'from' field")
	}

	return nil
}

// SubscribeOptions are options for Chat.Subscribe method.
type SubscribeOptions struct {
	ChatOptions
}

// SendOptions are options for Chat.Send.
type SendOptions struct {
	ChatOptions
}
