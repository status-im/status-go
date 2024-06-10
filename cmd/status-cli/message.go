package main

import (
	"bufio"
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

func (cli *StatusCLI) sendContactRequest(ctx context.Context, toID string) error {
	cli.logger.Info("send contact request, contact public key: ", toID)
	request := &requests.SendContactRequest{
		ID:      toID,
		Message: "Hello!",
	}
	resp, err := cli.messenger.SendContactRequest(ctx, request)
	cli.logger.Info("function SendContactRequest response.messages: ", resp.Messages())
	if err != nil {
		return err
	}

	return nil
}

func (cli *StatusCLI) sendContactRequestAcceptance(ctx context.Context, msgID string) error {
	cli.logger.Info("accept contact request, message ID: ", msgID)
	resp, err := cli.messenger.AcceptContactRequest(ctx, &requests.AcceptContactRequest{ID: types.Hex2Bytes(msgID)})
	if err != nil {
		return err
	}
	cli.logger.Info("function AcceptContactRequest response: ", resp.Messages())

	return nil
}

func (cli *StatusCLI) sendDirectMessage(ctx context.Context, text string) error {
	if len(cli.messenger.MutualContacts()) == 0 {
		return nil
	}
	chat := cli.messenger.Chat(cli.messenger.MutualContacts()[0].ID)
	cli.logger.Info("will send message to contact: ", chat.ID)

	clock, timestamp := chat.NextClockAndTimestamp(cli.messenger.GetTransport())
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	inputMessage.LocalChatID = chat.ID
	inputMessage.Clock = clock
	inputMessage.Timestamp = timestamp
	inputMessage.MessageType = protobuf.MessageType_ONE_TO_ONE
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = text

	resp, err := cli.messenger.SendChatMessage(ctx, inputMessage)
	if err != nil {
		return err
	}

	for _, message := range resp.Messages() {
		cli.logger.Infof("sent message: %v (ID=%v)", message.Text, message.ID)
	}

	return nil
}

func (cli *StatusCLI) retrieveMessagesLoop(ctx context.Context, tick time.Duration, msgCh chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	cli.logger.Info("retrieve messages...")

	for {
		select {
		case <-ticker.C:
			response, err := cli.messenger.RetrieveAll()
			if err != nil {
				cli.logger.Error("failed to retrieve raw messages", "err", err)
				continue
			}
			if response == nil {
				continue
			}
			for _, message := range response.Messages() {
				cli.logger.Infof("message received: %v (ID=%v)", message.Text, message.ID)
				if message.ContentType == protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_EVENT_SENT {
					msgCh <- message.ID
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// interactiveSendMessageLoop reads input from stdin and sends it as a direct message to the first mutual contact.
//
// If multiple CLIs are provided, it will send messages in a round-robin fashion:
// 1st input message will be from Alice, 2nd from Bob, 3rd from Alice, and so on.
func interactiveSendMessageLoop(ctx context.Context, clis ...*StatusCLI) {
	reader := bufio.NewReader(os.Stdin)
	i := -1
	n := len(clis)
	if n == 0 {
		slog.Error("at least 1 CLI needed")
		return
	}
	for {
		i++
		if i >= n {
			i = 0
		}
		cli := clis[i] // round robin cli selection

		if len(cli.messenger.MutualContacts()) == 0 {
			// waits for 1 second before trying again
			time.Sleep(1 * time.Second)
			continue
		}
		cli.logger.Info("Enter your message to send: (type 'quit' or 'q' to exit)")

		message, err := readInput(ctx, reader)
		if err != nil {
			if err == context.Canceled {
				return
			}
			cli.logger.Error("failed to read input", err)
			continue
		}
		message = strings.TrimSpace(message)
		if message == "quit" || message == "q" || strings.Contains(message, "\x03") {
			return
		}
		if message == "" {
			continue
		}
		if err = cli.sendDirectMessage(ctx, message); err != nil {
			cli.logger.Error("failed to send direct message: ", err)
			continue
		}
	}
}

// readInput reads input from the reader and respects context cancellation
func readInput(ctx context.Context, reader *bufio.Reader) (string, error) {
	inputCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Start a goroutine to read input
	go func() {
		input, err := reader.ReadString('\n')
		if err != nil {
			errCh <- err
			return
		}
		inputCh <- input
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case input := <-inputCh:
		return input, nil
	case err := <-errCh:
		return "", err
	}
}
