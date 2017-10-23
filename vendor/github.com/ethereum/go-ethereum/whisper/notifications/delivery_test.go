package notifications_test

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/message"
	"github.com/ethereum/go-ethereum/whisper/notifications"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

func TestDeliveryService(t *testing.T) {
	var delivery notifications.DeliveryService

	t.Logf("When unsubscribing from service no message should be received")
	{
		received := make(chan struct{}, 0)
		id := delivery.Subscribe(func(msg notifications.DeliveryState) {
			close(received)
		})

		delivery.Unsubscribe(id)
		delivery.SendRPCState(whisper.RPCMessageState{})

		case <-received:
			t.Fatalf("\t Should have successfully not recevied message notification")
		case <-time.After(5 * time.Millisecond):
			t.Logf("\t Should have successfully not recevied message notification")
		}
	}

	t.Logf("When subscribing for rpc whisper messages")
	{
		received := make(chan struct{}, 0)
		id := delivery.SubscribeForRPC(func(rpcmsg *whisper.RPCMessageState) {
			close(received)
		})

		delivery.SendRPCState(whisper.RPCMessageState{})
		delivery.Unsubscribe(id)

		select {
		case <-received:
			t.Logf("\t Should have successfully received rpc message notification")
		case <-time.After(5 * time.Millisecond):
			t.Fatalf("\t Should have successfully received rpc message notification")
		}
	}

	t.Logf("When subscribing for p2p whisper messages")
	{
		received := make(chan struct{}, 0)
		id := delivery.SubscribeForP2P(func(rpcmsg *whisper.P2PMessageState) {
			close(received)
		})

		delivery.SendP2PState(whisper.P2PMessageState{})
		delivery.Unsubscribe(id)

		select {
		case <-received:
			t.Logf("\t Should have successfully received p2p message notification")
		case <-time.After(5 * time.Millisecond):
			t.Fatalf("\t Should have successfully received p2p2 message notification")
		}
	}

	t.Logf("When subscribing for rpc message based on direction should only get message with direction value")
	{
		received := make(chan struct{}, 3)
		id := delivery.ByRPCDirection(message.IncomingMessage, func(msg *whisper.RPCMessageState) {
			received <- struct{}{}
		})

		delivery.SendRPCState(whisper.RPCMessageState{Direction: message.IncomingMessage})
		delivery.SendRPCState(whisper.RPCMessageState{Direction: message.OutgoingMessage})
		delivery.SendP2PState(whisper.P2PMessageState{Direction: message.IncomingMessage})
		delivery.Unsubscribe(id)

		if len(received) != 1 {
			t.Fatalf("\t Should have successfully recevied only one rpc message notification")
		}
		t.Logf("\t Should have successfully recevied only one rpc message notification")
	}

	t.Logf("When subscribing p2p message based on direction should only get message with direction value")
	{
		received := make(chan struct{}, 3)
		id := delivery.ByP2PDirection(message.IncomingMessage, func(msg *whisper.P2PMessageState) {
			received <- struct{}{}
		})

		delivery.SendRPCState(whisper.RPCMessageState{Direction: message.IncomingMessage})
		delivery.SendRPCState(whisper.RPCMessageState{Direction: message.OutgoingMessage})
		delivery.SendP2PState(whisper.P2PMessageState{Direction: message.IncomingMessage})
		delivery.Unsubscribe(id)

		if len(received) != 1 {
			t.Fatalf("\t Should have successfully recevied only one rpc message notification")
		}
		t.Logf("\t Should have successfully recevied only one rpc message notification")
	}

	t.Logf("When subscribing for rpc message based on status should only get message with status")
	{
		received := make(chan struct{}, 3)
		id := delivery.ByRPCStatus(message.RejectedStatus, func(msg *whisper.RPCMessageState) {
			received <- struct{}{}
		})

		delivery.SendRPCState(whisper.RPCMessageState{Status: message.PendingStatus})
		delivery.SendRPCState(whisper.RPCMessageState{Status: message.RejectedStatus})
		delivery.SendP2PState(whisper.P2PMessageState{Status: message.RejectedStatus})
		delivery.Unsubscribe(id)

		if len(received) != 1 {
			t.Fatalf("\t Should have successfully recevied only one rpc message notification")
		}
		t.Logf("\t Should have successfully recevied only one rpc message notification")
	}

	t.Logf("When subscribing for p2p message based on status should only get message with status")
	{
		received := make(chan struct{}, 3)
		id := delivery.ByP2PStatus(message.RejectedStatus, func(msg *whisper.P2PMessageState) {
			received <- struct{}{}
		})

		delivery.SendRPCState(whisper.RPCMessageState{Status: message.PendingStatus})
		delivery.SendRPCState(whisper.RPCMessageState{Status: message.RejectedStatus})
		delivery.SendP2PState(whisper.P2PMessageState{Status: message.RejectedStatus})
		delivery.Unsubscribe(id)

		if len(received) != 1 {
			t.Fatalf("\t Should have successfully recevied only one rpc message notification")
		}
		t.Logf("\t Should have successfully recevied only one rpc message notification")
	}
}
