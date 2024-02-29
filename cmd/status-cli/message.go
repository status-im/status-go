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

func sendContactRequest(cCtx *cli.Context, from *StatusCLI, toId string) error {
	from.logger.Info("send contact request, contact public key: ", toId)
	request := &requests.SendContactRequest{
		ID:      toId,
		Message: "Hello!",
	}
	resp, err := from.messenger.SendContactRequest(cCtx.Context, request)
	from.logger.Info("function SendContactRequest response.messages: ", resp.Messages())
	if err != nil {
		return err
	}

	return nil
}

func sendContactRequestAcceptance(cCtx *cli.Context, from *StatusCLI, msgID string) error {
	from.logger.Info("accept contact request, message ID: ", msgID)
	resp, err := from.messenger.AcceptContactRequest(cCtx.Context, &requests.AcceptContactRequest{ID: types.Hex2Bytes(msgID)})
	if err != nil {
		return err
	}
	from.logger.Info("function AcceptContactRequest response: ", resp.Messages())

	return nil
}

func sendDirectMessage(from *StatusCLI, text string, ctx context.Context) error {
	if len(from.messenger.MutualContacts()) == 0 {
		return nil
	}
	chat := from.messenger.Chat(from.messenger.MutualContacts()[0].ID)
	from.logger.Info("send message to contact: ", chat.ID)

	clock, timestamp := chat.NextClockAndTimestamp(from.messenger.GetTransport())
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	inputMessage.LocalChatID = chat.ID
	inputMessage.Clock = clock
	inputMessage.Timestamp = timestamp
	inputMessage.MessageType = protobuf.MessageType_ONE_TO_ONE
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = text

	_, err := from.messenger.SendChatMessage(ctx, inputMessage)
	if err != nil {
		return err
	}

	return nil
}

func retrieveMessagesLoop(cli *StatusCLI, tick time.Duration, msgCh chan string, ctx context.Context, wg *sync.WaitGroup) {
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
			if response != nil && len(response.Messages()) != 0 {
				for _, message := range response.Messages() {
					cli.logger.Info("receive message: ", message.Text)
					if message.ContentType == protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_EVENT_SENT {
						msgCh <- message.ID
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func sendMessageLoop(cli *StatusCLI, tick time.Duration, ctx context.Context, wg *sync.WaitGroup, sem chan struct{}, cancel context.CancelFunc) {
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

			err = sendDirectMessage(cli, message, ctx)
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
