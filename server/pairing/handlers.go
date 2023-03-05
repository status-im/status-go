package pairing

import (
	"bytes"
	"crypto/rand"
	"io"
	"net/http"

	"github.com/btcsuite/btcutil/base58"
	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/common"
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

	// Session names
	sessionChallenge = "challenge"
	sessionBlocked   = "blocked"
)

func handleReceiveAccount(ps *Server) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionPairingAccount})
	logger := ps.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingAccount})
			logger.Error("handleReceiveAccount io.ReadAll(r.Body)", zap.Error(err))
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionPairingAccount})

		err = ps.PayloadManager.Receive(payload)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventProcessError, Error: err.Error(), Action: ActionPairingAccount})
			logger.Error("ps.PayloadManager.Receive(payload)", zap.Error(err))
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventProcessSuccess, Action: ActionPairingAccount})
	}
}

func handleReceiveInstallation(ps *Server) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionPairingInstallation})
	logger := ps.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingInstallation})
			logger.Error("handleReceiveInstallation io.ReadAll(r.Body)", zap.Error(err))
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionPairingInstallation})

		err = ps.installationPayloadManager.Receive(payload)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventProcessError, Error: err.Error(), Action: ActionPairingInstallation})
			logger.Error("ps.installationPayloadManager.Receive(payload)", zap.Error(err))
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventProcessSuccess, Action: ActionPairingInstallation})
	}
}

func handleParingSyncDeviceReceive(ps *Server) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionSyncDevice})
	logger := ps.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionSyncDevice})
			logger.Error("handleParingSyncDeviceReceive io.ReadAll(r.Body)", zap.Error(err))
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionSyncDevice})

		err = ps.rawMessagePayloadManager.Receive(payload)
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventProcessError, Error: err.Error(), Action: ActionSyncDevice})
			logger.Error("ps.rawMessagePayloadManager.Receive(payload)", zap.Error(err))
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventProcessSuccess, Action: ActionSyncDevice})
	}
}

func handleSendAccount(ps *Server) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionPairingAccount})
	logger := ps.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, err := w.Write(ps.PayloadManager.ToSend())
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingAccount})
			logger.Error("w.Write(ps.PayloadManager.ToSend())", zap.Error(err))
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionPairingAccount})

		ps.PayloadManager.LockPayload()
	}
}

func handleSendInstallation(ps *Server) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionPairingInstallation})
	logger := ps.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		err := ps.installationPayloadManager.Mount()
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingInstallation})
			logger.Error("ps.installationPayloadManager.Mount()", zap.Error(err))
			return
		}

		_, err = w.Write(ps.installationPayloadManager.ToSend())
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionPairingInstallation})
			logger.Error("w.Write(ps.installationPayloadManager.ToSend())", zap.Error(err))
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionPairingInstallation})

		ps.installationPayloadManager.LockPayload()
	}
}

func handlePairingSyncDeviceSend(ps *Server) http.HandlerFunc {
	signal.SendLocalPairingEvent(Event{Type: EventConnectionSuccess, Action: ActionSyncDevice})
	logger := ps.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")

		err := ps.rawMessagePayloadManager.Mount()
		if err != nil {
			// maybe better to use a new event type here instead of EventTransferError?
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionSyncDevice})
			logger.Error("ps.rawMessagePayloadManager.Mount()", zap.Error(err))
			return
		}

		_, err = w.Write(ps.rawMessagePayloadManager.ToSend())
		if err != nil {
			signal.SendLocalPairingEvent(Event{Type: EventTransferError, Error: err.Error(), Action: ActionSyncDevice})
			logger.Error("w.Write(ps.rawMessagePayloadManager.ToSend())", zap.Error(err))
			return
		}
		signal.SendLocalPairingEvent(Event{Type: EventTransferSuccess, Action: ActionSyncDevice})

		ps.rawMessagePayloadManager.LockPayload()
	}
}

func challengeMiddleware(ps *Server, next http.Handler) http.HandlerFunc {
	logger := ps.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := ps.cookieStore.Get(r, sessionChallenge)
		if err != nil {
			logger.Error("ps.cookieStore.Get(r, pairingStoreChallenge)", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		blocked, ok := s.Values[sessionBlocked].(bool)
		if ok && blocked {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		// If the request header doesn't include a challenge don't punish the client, just throw a 403
		pc := r.Header.Get(sessionChallenge)
		if pc == "" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		c, err := common.Decrypt(base58.Decode(pc), ps.ek)
		if err != nil {
			logger.Error("c, err := common.Decrypt(rc, ps.ek)", zap.Error(err))
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		// If the challenge is not in the session store don't punish the client, just throw a 403
		challenge, ok := s.Values[sessionChallenge].([]byte)
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		// Only if we have both a challenge in the session store and in the request header
		// do we entertain blocking the client. Because then we know someone is trying to be sneaky.
		if !bytes.Equal(c, challenge) {
			s.Values[sessionBlocked] = true
			err = s.Save(r, w)
			if err != nil {
				logger.Error("err = s.Save(r, w)", zap.Error(err))
			}

			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func handlePairingChallenge(ps *Server) http.HandlerFunc {
	logger := ps.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := ps.cookieStore.Get(r, sessionChallenge)
		if err != nil {
			logger.Error("ps.cookieStore.Get(r, pairingStoreChallenge)", zap.Error(err))
			return
		}

		var challenge []byte
		challenge, ok := s.Values[sessionChallenge].([]byte)
		if !ok {
			challenge = make([]byte, 64)
			_, err = rand.Read(challenge)
			if err != nil {
				logger.Error("_, err = rand.Read(auth)", zap.Error(err))
				return
			}

			s.Values[sessionChallenge] = challenge
			err = s.Save(r, w)
			if err != nil {
				logger.Error("err = s.Save(r, w)", zap.Error(err))
				return
			}
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		_, err = w.Write(challenge)
		if err != nil {
			logger.Error("_, err = w.Write(challenge)", zap.Error(err))
			return
		}
	}
}
