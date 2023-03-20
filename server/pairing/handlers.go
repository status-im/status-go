package pairing

import (
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/status-im/status-go/signal"
)

const (
	// Handler routes for pairing
	pairingBase                = "/pairing"
	pairingChallenge           = pairingBase + "/challenge"
	pairingSendAccount         = pairingBase + "/sendAccount"
	pairingReceiveAccount      = pairingBase + "/receiveAccount"
	pairingSendSyncDevice      = pairingBase + "/sendSyncDevice"
	pairingReceiveSyncDevice   = pairingBase + "/receiveSyncDevice"
	pairingSendInstallation    = pairingBase + "/sendInstallation"
	pairingReceiveInstallation = pairingBase + "/receiveInstallation"
)

// Account handling

func handleReceiveAccount(hs HandlerServer, pr PayloadReceiver) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionPairingAccount})
	logger := hs.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingAccount})
			logger.Error("handleReceiveAccount io.ReadAll(r.Body)", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionPairingAccount})

		err = pr.Receive(payload)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventProcessError, Error: err.Error(), Action: ActionPairingAccount})
			logger.Error("handleReceiveAccount pr.Receive(payload)", zap.Error(err), zap.Binary("payload", payload))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventProcessSuccess, Action: ActionPairingAccount})
	}
}

func handleSendAccount(hs HandlerServer, pm PayloadMounter) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionPairingAccount})
	logger := hs.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		err := pm.Mount()
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingAccount})
			logger.Error("handleSendAccount pm.Mount()", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		_, err = w.Write(pm.ToSend())
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingAccount})
			logger.Error("handleSendAccount w.Write(pm.ToSend())", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionPairingAccount})

		pm.LockPayload()
	}
}

// Device sync handling

func handleParingSyncDeviceReceive(hs HandlerServer, pr PayloadReceiver) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionSyncDevice})
	logger := hs.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionSyncDevice})
			logger.Error("handleParingSyncDeviceReceive io.ReadAll(r.Body)", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionSyncDevice})

		err = pr.Receive(payload)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventProcessError, Error: err.Error(), Action: ActionSyncDevice})
			logger.Error("handleParingSyncDeviceReceive pr.Receive(payload)", zap.Error(err), zap.Binary("payload", payload))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventProcessSuccess, Action: ActionSyncDevice})
	}
}

func handlePairingSyncDeviceSend(hs HandlerServer, pm PayloadMounter) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionSyncDevice})
	logger := hs.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")

		err := pm.Mount()
		if err != nil {
			// maybe better to use a new event type here instead of EventTransferError?
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionSyncDevice})
			logger.Error("handlePairingSyncDeviceSend pm.Mount()", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		_, err = w.Write(pm.ToSend())
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionSyncDevice})
			logger.Error("handlePairingSyncDeviceSend w.Write(pm.ToSend())", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionSyncDevice})

		pm.LockPayload()
	}
}

// Installation data handling

func handleReceiveInstallation(hs HandlerServer, pmr PayloadMounterReceiver) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionPairingInstallation})
	logger := hs.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingInstallation})
			logger.Error("handleReceiveInstallation io.ReadAll(r.Body)", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionPairingInstallation})

		err = pmr.Receive(payload)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventProcessError, Error: err.Error(), Action: ActionPairingInstallation})
			logger.Error("handleReceiveInstallation pmr.Receive(payload)", zap.Error(err), zap.Binary("payload", payload))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventProcessSuccess, Action: ActionPairingInstallation})
	}
}

func handleSendInstallation(hs HandlerServer, pmr PayloadMounterReceiver) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionPairingInstallation})
	logger := hs.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		err := pmr.Mount()
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingInstallation})
			logger.Error("handleSendInstallation pmr.Mount()", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		_, err = w.Write(pmr.ToSend())
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingInstallation})
			logger.Error("handleSendInstallation w.Write(pmr.ToSend())", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionPairingInstallation})

		pmr.LockPayload()
	}
}

// Challenge middleware and handling

func middlewareChallenge(cg *ChallengeGiver, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ce := cg.checkChallengeResponse(w, r)
		if ce != nil {
			http.Error(w, ce.Text, ce.HttpCode)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func handlePairingChallenge(cg *ChallengeGiver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		challenge, ce := cg.getChallenge(w, r)
		if ce != nil {
			http.Error(w, ce.Text, ce.HttpCode)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		_, err := w.Write(challenge)
		if err != nil {
			cg.logger.Error("handlePairingChallenge: _, err = w.Write(challenge)", zap.Error(err))
			return
		}
	}
}
