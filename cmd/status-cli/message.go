package main

import (
	"bufio"
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"

	"github.com/urfave/cli/v2"
)

func (cli *StatusCLI) sendContactRequest(cCtx *cli.Context, toID string) error {
	cli.logger.Info("send contact request, contact public key: ", toID)
	request := &requests.SendContactRequest{
		ID:      toID,
		Message: "Hello!",
	}
	resp, err := cli.messenger.SendContactRequest(cCtx.Context, request)
	cli.logger.Info("function SendContactRequest response.messages: ", resp.Messages())
	if err != nil {
		return err
	}

	return nil
}

func (cli *StatusCLI) sendContactRequestAcceptance(cCtx *cli.Context, msgID string) error {
	cli.logger.Info("accept contact request, message ID: ", msgID)
	resp, err := cli.messenger.AcceptContactRequest(cCtx.Context, &requests.AcceptContactRequest{ID: types.Hex2Bytes(msgID)})
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

func (cli *StatusCLI) sendMessageLoop(ctx context.Context, tick time.Duration, wg *sync.WaitGroup, sem chan struct{}, cancel context.CancelFunc) {
	defer wg.Done()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-ticker.C:
			if len(cli.messenger.MutualContacts()) == 0 {
				continue
			}
			sem <- struct{}{}
			cli.logger.Info("Enter your message to send: (type 'quit' or 'q' to exit)")
			message, err := reader.ReadString('\n')
			if err != nil {
				<-sem
				cli.logger.Error("failed to read input", err)
				continue
			}

			message = strings.TrimSpace(message)
			if message == "quit" || message == "q" || strings.Contains(message, "\x03") {
				cancel()
				<-sem
				return
			}
			if message == "" {
				<-sem
				continue
			}

			err = cli.sendDirectMessage(ctx, message)
			time.Sleep(WaitingInterval)
			<-sem
			if err != nil {
				cli.logger.Error("failed to send direct message: ", err)
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}

func (cli *StatusCLI) sendMessageLoop2(ctx context.Context) {
	reader := bufio.NewReader(os.Stdin)
	for {
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
