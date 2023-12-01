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

func handleReceiveAccount(logger *zap.Logger, pr PayloadReceiver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionPairingAccount})
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

func handleSendAccount(logger *zap.Logger, pm PayloadMounter, beforeSending func()) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionPairingAccount})
		w.Header().Set("Content-Type", "application/octet-stream")
		err := pm.Mount()
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingAccount})
			logger.Error("handleSendAccount pm.Mount()", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		beforeSending()
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

func handleParingSyncDeviceReceive(logger *zap.Logger, pr PayloadReceiver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionSyncDevice})
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

func handlePairingSyncDeviceSend(logger *zap.Logger, pm PayloadMounter, beforeSending func()) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionSyncDevice})
		w.Header().Set("Content-Type", "application/octet-stream")

		err := pm.Mount()
		if err != nil {
			// maybe better to use a new event type here instead of EventTransferError?
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionSyncDevice})
			logger.Error("handlePairingSyncDeviceSend pm.Mount()", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		beforeSending()
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

func handleReceiveInstallation(logger *zap.Logger, pmr PayloadMounterReceiver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionPairingInstallation})
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

func handleSendInstallation(logger *zap.Logger, pmr PayloadMounterReceiver, beforeSending func()) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionPairingInstallation})
		w.Header().Set("Content-Type", "application/octet-stream")
		err := pmr.Mount()
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingInstallation})
			logger.Error("handleSendInstallation pmr.Mount()", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		beforeSending()
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
		err := cg.checkChallengeResponse(w, r)
		if err != nil {
			if cErr, ok := err.(*ChallengeError); ok {
				http.Error(w, cErr.Text, cErr.HTTPCode)
				return
			}
			cg.logger.Error("failed to checkChallengeResponse in middlewareChallenge", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func handlePairingChallenge(cg *ChallengeGiver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		challenge, err := cg.getChallenge(w, r)
		if err != nil {
			if cErr, ok := err.(*ChallengeError); ok {
				http.Error(w, cErr.Text, cErr.HTTPCode)
				return
			}
			cg.logger.Error("failed to getChallenge in handlePairingChallenge", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		_, err = w.Write(challenge)
		if err != nil {
			cg.logger.Error("failed to Write(challenge) in handlePairingChallenge", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
	}
}
