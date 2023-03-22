package pairing

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net"
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
	HTTPCode int
}

func (ce *ChallengeError) Error() string {
	return fmt.Sprintf("%s : %d", ce.Text, ce.HTTPCode)
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
	authedIP    net.IP
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

func (cg *ChallengeGiver) getIP(r *http.Request) (net.IP, error) {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		cg.logger.Error("getIP: h, _, err := net.SplitHostPort(r.RemoteAddr)", zap.Error(err), zap.String("r.RemoteAddr", r.RemoteAddr))
		return nil, &ChallengeError{"error", http.StatusInternalServerError}
	}
	return net.ParseIP(h), nil
}

func (cg *ChallengeGiver) registerClientIP(r *http.Request) error {
	IP, err := cg.getIP(r)
	if err != nil {
		return err
	}
	cg.authedIP = IP
	return nil
}

func (cg *ChallengeGiver) validateClientIP(r *http.Request) error {
	// If we haven't registered yet register the IP
	if cg.authedIP == nil || len(cg.authedIP) == 0 {
		err := cg.registerClientIP(r)
		if err != nil {
			return err
		}
	}

	// Then compare the current req RemoteIP with the authed IP
	IP, err := cg.getIP(r)
	if err != nil {
		return err
	}

	if !cg.authedIP.Equal(IP) {
		cg.logger.Error(
			"request RemoteAddr does not match authedIP: expected '%s', received '%s'",
			zap.String("expected", cg.authedIP.String()),
			zap.String("received", IP.String()),
		)
		return &ChallengeError{"forbidden", http.StatusForbidden}
	}
	return nil
}

func (cg *ChallengeGiver) getSession(r *http.Request) (*sessions.Session, error) {
	s, err := cg.cookieStore.Get(r, sessionChallenge)
	if err != nil {
		cg.logger.Error("checkChallengeResponse: cg.cookieStore.Get(r, sessionChallenge)", zap.Error(err), zap.String("sessionChallenge", sessionChallenge))
		return nil, &ChallengeError{"error", http.StatusInternalServerError}
	}
	return s, nil
}

func (cg *ChallengeGiver) generateNewChallenge(s *sessions.Session, w http.ResponseWriter, r *http.Request) ([]byte, error) {
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

func (cg *ChallengeGiver) block(s *sessions.Session, w http.ResponseWriter, r *http.Request) error {
	s.Values[sessionBlocked] = true
	err := s.Save(r, w)
	if err != nil {
		cg.logger.Error("block: err = s.Save(r, w)", zap.Error(err))
		return &ChallengeError{"error", http.StatusInternalServerError}
	}

	return &ChallengeError{"forbidden", http.StatusForbidden}
}

func (cg *ChallengeGiver) checkChallengeResponse(w http.ResponseWriter, r *http.Request) error {
	err := cg.validateClientIP(r)
	if err != nil {
		return err
	}

	s, err := cg.getSession(r)
	if err != nil {
		return err
	}

	blocked, ok := s.Values[sessionBlocked].(bool)
	if ok && blocked {
		return &ChallengeError{"forbidden", http.StatusForbidden}
	}

	// If the request header doesn't include a challenge don't punish the client, just throw a 403
	clientChallengeResp := r.Header.Get(sessionChallenge)
	if clientChallengeResp == "" {
		return &ChallengeError{"forbidden", http.StatusForbidden}
	}

	dcr, err := cg.encryptor.decryptPlain(base58.Decode(clientChallengeResp))
	if err != nil {
		cg.logger.Error("checkChallengeResponse: cg.encryptor.decryptPlain(base58.Decode(clientChallengeResp))", zap.Error(err), zap.String("clientChallengeResp", clientChallengeResp))
		return &ChallengeError{"error", http.StatusInternalServerError}
	}

	// If the challenge is not in the session store don't punish the client, just throw a 403
	challenge, ok := s.Values[sessionChallenge].([]byte)
	if !ok {
		return &ChallengeError{"forbidden", http.StatusForbidden}
	}

	// Only if we have both a challenge in the session store and in the request header
	// do we entertain blocking the client. Because then we know someone is trying to be sneaky.
	if !bytes.Equal(dcr, challenge) {
		return cg.block(s, w, r)
	}

	// If every is ok, generate a new challenge for the next req
	_, err = cg.generateNewChallenge(s, w, r)
	return err
}

func (cg *ChallengeGiver) getChallenge(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	err := cg.validateClientIP(r)
	if err != nil {
		return nil, err
	}

	s, err := cg.getSession(r)
	if err != nil {
		return nil, err
	}

	challenge, ok := s.Values[sessionChallenge].([]byte)
	if !ok {
		challenge, err = cg.generateNewChallenge(s, w, r)
		if err != nil {
			return nil, err
		}
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
