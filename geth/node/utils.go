package node

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	osSignal "os/signal"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethmessage "github.com/ethereum/go-ethereum/common/message"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/whisper/notifications"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
)

// HaltOnPanic recovers from panic, logs issue, sends upward notification, and exits
func HaltOnPanic() {
	if r := recover(); r != nil {
		err := fmt.Errorf("%v: %v", ErrNodeRunFailure, r)

		// send signal up to native app
		signal.Send(signal.Envelope{
			Type: signal.EventNodeCrashed,
			Event: signal.NodeCrashEvent{
				Error: err.Error(),
			},
		})

		common.Fatalf(err) // os.exit(1) is called internally
	}
}

// HaltOnInterruptSignal stops node and panics if you press Ctrl-C enough times
func HaltOnInterruptSignal(nodeManager *NodeManager) {
	sigc := make(chan os.Signal, 1)
	osSignal.Notify(sigc, os.Interrupt)
	defer osSignal.Stop(sigc)
	<-sigc
	if nodeManager.node == nil {
		return
	}
	log.Info("Got interrupt, shutting down...")
	go nodeManager.node.Stop() // nolint: errcheck
	for i := 3; i > 0; i-- {
		<-sigc
		if i > 1 {
			log.Info(fmt.Sprintf("Already shutting down, interrupt %d more times for panic.", i-1))
		}
	}
	panic("interrupted!")
}

// logMessageStat logs stat derived from provided DeliveryState.
func logMessageStat(state notifications.DeliveryState) {
	stat := generateMessageStat(state)
	statdata, err := json.Marshal(stat)
	if err != nil {
		log.Warn("failed to marshal common.MessageStat", "err", err)
		return
	}

	encodedStat := base64.StdEncoding.EncodeToString(statdata)
	log.Info(fmt.Sprintf("%s : %s : %s : %s : %+s", params.MessageStatHeader, stat.Protocol, stat.Type, stat.Status, encodedStat))
}

// generateMessageStat returns a common.MessageStat instance after retrieving
// data from DeliveryState.
func generateMessageStat(state notifications.DeliveryState) common.MessageStat {
	var stat common.MessageStat
	var payload []byte
	var from, to string

	if state.IsP2P {
		if state.P2P.Direction == gethmessage.IncomingMessage {
			if state.P2P.Received != nil {
				payload = state.P2P.Received.Payload

				if state.P2P.Received.Src != nil {
					from = gethcommon.ToHex(crypto.FromECDSAPub(state.P2P.Received.Src))
				}

				if state.P2P.Received.Dst != nil {
					to = gethcommon.ToHex(crypto.FromECDSAPub(state.P2P.Received.Dst))
				}
			}
		}

		if state.P2P.Direction == gethmessage.OutgoingMessage {
			from = state.P2P.Source.Sig

			if len(state.P2P.Source.PublicKey) == 0 {
				to = string(state.P2P.Source.PublicKey)
			} else {
				to = state.P2P.Source.TargetPeer
			}
		}

		stat.Protocol = "P2P"
		stat.Payload = payload
		stat.FromDevice = from
		stat.ToDevice = to
		stat.Source = state.P2P.Source
		stat.RejectionReason = state.P2P.Reason
		stat.Envelope = state.P2P.Envelope.Data
		stat.Status = state.P2P.Status.String()
		stat.Type = state.P2P.Direction.String()
		stat.Hash = state.P2P.Envelope.Hash().String()
		stat.TimeSent = state.P2P.Envelope.Expiry - state.P2P.Envelope.TTL

	} else {
		if state.RPC.Direction == gethmessage.IncomingMessage {
			if state.RPC.Received != nil {
				payload = state.RPC.Received.Payload

				if state.RPC.Received.Src != nil {
					from = gethcommon.ToHex(crypto.FromECDSAPub(state.RPC.Received.Src))
				}

				if state.RPC.Received.Dst != nil {
					to = gethcommon.ToHex(crypto.FromECDSAPub(state.RPC.Received.Dst))
				}
			}
		}

		if state.RPC.Direction == gethmessage.OutgoingMessage {
			from = state.RPC.Source.Sig

			if len(state.RPC.Source.PublicKey) == 0 {
				to = string(state.RPC.Source.PublicKey)
			} else {
				to = state.RPC.Source.TargetPeer
			}
		}

		stat.Protocol = "RPC"
		stat.Payload = payload
		stat.FromDevice = from
		stat.ToDevice = to
		stat.Source = state.RPC.Source
		stat.RejectionReason = state.RPC.Reason
		stat.Envelope = state.RPC.Envelope.Data
		stat.Status = state.RPC.Status.String()
		stat.Type = state.RPC.Direction.String()
		stat.Hash = state.RPC.Envelope.Hash().String()
		stat.TimeSent = state.RPC.Envelope.Expiry - state.RPC.Envelope.TTL
	}

	return stat
}
