package pairing

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"net/http"

	"github.com/btcsuite/btcutil/base58"
	"github.com/gorilla/sessions"
	"go.uber.org/zap"
)

type ChallengeError struct {
	Text     string
	HttpCode int
}

func (ce *ChallengeError) Error() string {
	return fmt.Sprintf("%s : %d", ce.Text, ce.HttpCode)
}

func makeCookieStore() (*sessions.CookieStore, error) {
	auth := make([]byte, 64)
	_, err := rand.Read(auth)
	if err != nil {
		return nil, err
	}

	enc := make([]byte, 32)
	_, err = rand.Read(enc)
	if err != nil {
		return nil, err
	}

	return sessions.NewCookieStore(auth, enc), nil
}

type ChallengeGiver struct {
	cookieStore *sessions.CookieStore
	encryptor   *PayloadEncryptor
	logger      *zap.Logger
}

func NewChallengeGiver(e *PayloadEncryptor, logger *zap.Logger) (*ChallengeGiver, error) {
	cs, err := makeCookieStore()
	if err != nil {
		return nil, err
	}

	return &ChallengeGiver{
		cookieStore: cs,
		encryptor:   e,
		logger:      logger,
	}, nil
}

func (cg *ChallengeGiver) handleChallengeResponse(w http.ResponseWriter, r *http.Request) *ChallengeError {
	s, err := cg.cookieStore.Get(r, sessionChallenge)
	if err != nil {
		cg.logger.Error("hs.GetCookieStore().Get(r, sessionChallenge)", zap.Error(err))
		return &ChallengeError{"error", http.StatusInternalServerError}
	}

	blocked, ok := s.Values[sessionBlocked].(bool)
	if ok && blocked {
		return &ChallengeError{"forbidden", http.StatusForbidden}
	}

	// If the request header doesn't include a challenge don't punish the client, just throw a 403
	pc := r.Header.Get(sessionChallenge)
	if pc == "" {
		return &ChallengeError{"forbidden", http.StatusForbidden}
	}

	c, err := cg.encryptor.decryptPlain(base58.Decode(pc))
	if err != nil {
		cg.logger.Error("c, err := hs.DecryptPlain(rc, hs.ek)", zap.Error(err))
		return &ChallengeError{"error", http.StatusInternalServerError}
	}

	// If the challenge is not in the session store don't punish the client, just throw a 403
	challenge, ok := s.Values[sessionChallenge].([]byte)
	if !ok {
		return &ChallengeError{"forbidden", http.StatusForbidden}
	}

	// Only if we have both a challenge in the session store and in the request header
	// do we entertain blocking the client. Because then we know someone is trying to be sneaky.
	if !bytes.Equal(c, challenge) {
		s.Values[sessionBlocked] = true
		err = s.Save(r, w)
		if err != nil {
			cg.logger.Error("err = s.Save(r, w)", zap.Error(err))
		}

		return &ChallengeError{"forbidden", http.StatusForbidden}
	}
	return nil
}

func (cg *ChallengeGiver) challenge(w http.ResponseWriter, r *http.Request) *ChallengeError {
	s, err := cg.cookieStore.Get(r, sessionChallenge)
	if err != nil {
		cg.logger.Error("hs.GetCookieStore().Get(r, sessionChallenge)", zap.Error(err))
		return &ChallengeError{"error", http.StatusInternalServerError}
	}

	challenge, ok := s.Values[sessionChallenge].([]byte)
	if !ok {
		challenge = make([]byte, 64)
		_, err = rand.Read(challenge)
		if err != nil {
			cg.logger.Error("_, err = rand.Read(challenge)", zap.Error(err))
			return &ChallengeError{"error", http.StatusInternalServerError}
		}

		s.Values[sessionChallenge] = challenge
		err = s.Save(r, w)
		if err != nil {
			cg.logger.Error("err = s.Save(r, w)", zap.Error(err))
			return &ChallengeError{"error", http.StatusInternalServerError}
		}
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, err = w.Write(challenge)
	if err != nil {
		cg.logger.Error("_, err = w.Write(challenge)", zap.Error(err))
	}
	return nil
}

func (s *BaseServer) GetCookieStore() *sessions.CookieStore {
	return s.cookieStore
}

func (s *BaseServer) DecryptPlain(data []byte) ([]byte, error) {
	return s.encryptor.decryptPlain(data)
}

type ChallengeTaker struct {
	encryptor *PayloadEncryptor
}
