package pairing

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/btcsuite/btcutil/base58"
	"github.com/gorilla/sessions"
	"go.uber.org/zap"
)

const (
	// Session names
	sessionChallenge = "challenge"
	sessionBlocked   = "blocked"
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

// ChallengeGiver is responsible for generating challenges and checking challenge responses
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
		encryptor:   e.Renew(),
		logger:      logger,
	}, nil
}

func (cg *ChallengeGiver) getSession(r *http.Request) (*sessions.Session, *ChallengeError) {
	s, err := cg.cookieStore.Get(r, sessionChallenge)
	if err != nil {
		cg.logger.Error("checkChallengeResponse: cg.cookieStore.Get(r, sessionChallenge)", zap.Error(err), zap.String("sessionChallenge", sessionChallenge))
		return nil, &ChallengeError{"error", http.StatusInternalServerError}
	}
	return s, nil
}

func (cg *ChallengeGiver) generateNewChallenge(s *sessions.Session, w http.ResponseWriter, r *http.Request) ([]byte, *ChallengeError) {
	challenge := make([]byte, 64)
	_, err := rand.Read(challenge)
	if err != nil {
		cg.logger.Error("regenerateNewChallenge: _, err = rand.Read(challenge)", zap.Error(err))
		return nil, &ChallengeError{"error", http.StatusInternalServerError}
	}

	s.Values[sessionChallenge] = challenge
	err = s.Save(r, w)
	if err != nil {
		cg.logger.Error("regenerateNewChallenge: err = s.Save(r, w)", zap.Error(err))
		return nil, &ChallengeError{"error", http.StatusInternalServerError}
	}

	return challenge, nil
}

func (cg *ChallengeGiver) block(s *sessions.Session, w http.ResponseWriter, r *http.Request) *ChallengeError {
	s.Values[sessionBlocked] = true
	err := s.Save(r, w)
	if err != nil {
		cg.logger.Error("block: err = s.Save(r, w)", zap.Error(err))
		return &ChallengeError{"error", http.StatusInternalServerError}
	}

	return &ChallengeError{"forbidden", http.StatusForbidden}
}

func (cg *ChallengeGiver) checkChallengeResponse(w http.ResponseWriter, r *http.Request) *ChallengeError {
	s, ce := cg.getSession(r)
	if ce != nil {
		return ce
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
		cg.logger.Error("checkChallengeResponse: cg.encryptor.decryptPlain(base58.Decode(pc))", zap.Error(err), zap.String("pc", pc))
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
		return cg.block(s, w, r)
	}

	// If every is ok, regenerate the challenge
	_, ce = cg.generateNewChallenge(s, w, r)
	return ce
}

func (cg *ChallengeGiver) getChallenge(w http.ResponseWriter, r *http.Request) ([]byte, *ChallengeError) {
	s, ce := cg.getSession(r)
	if ce != nil {
		return nil, ce
	}

	challenge, ok := s.Values[sessionChallenge].([]byte)
	if !ok {
		challenge, ce = cg.generateNewChallenge(s, w, r)
	}
	return challenge, nil
}

// ChallengeTaker is responsible for storing and performing server challenges
type ChallengeTaker struct {
	encryptor       *PayloadEncryptor
	serverChallenge []byte
}

func NewChallengeTaker(e *PayloadEncryptor) *ChallengeTaker {
	return &ChallengeTaker{
		encryptor: e.Renew(),
	}
}

func (ct *ChallengeTaker) SetChallenge(resp *http.Response) error {
	challenge, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	ct.serverChallenge = challenge
	return nil
}

func (ct *ChallengeTaker) DoChallenge(req *http.Request) error {
	if ct.serverChallenge != nil {
		ec, err := ct.encryptor.encryptPlain(ct.serverChallenge)
		if err != nil {
			return err
		}

		req.Header.Set(sessionChallenge, base58.Encode(ec))
	}
	return nil
}
