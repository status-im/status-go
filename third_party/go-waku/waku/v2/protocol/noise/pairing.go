package noise

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"sync"

	n "github.com/waku-org/go-noise"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

var ErrPairingTimeout = errors.New("pairing has timed out")

type Sender interface {
	Publish(ctx context.Context, contentTopic string, payload *n.PayloadV2) error
}

type Receiver interface {
	// Subscribe will return a channel to obtain next message received in a content topic
	Subscribe(ctx context.Context, contentTopic string) <-chan *pb.WakuMessage
}

type Pairing struct {
	sync.RWMutex
	ContentTopic         string
	msgCh                <-chan *pb.WakuMessage
	randomFixLenVal      []byte
	myCommittedStaticKey []byte
	params               PairingParameters
	handshake            *n.Handshake
	authCode             string
	authCodeEmitted      chan string
	authCodeConfirmed    chan bool
	messenger            NoiseMessenger
	logger               *zap.Logger

	started   bool
	completed bool
	hsResult  *n.HandshakeResult
}

type PairingParameterOption func(*PairingParameters) error

func WithInitiatorParameters(qrString string, qrMessageNametag n.MessageNametag) PairingParameterOption {
	return func(params *PairingParameters) error {
		params.initiator = true
		qr, err := StringToQR(qrString)
		if err != nil {
			return err
		}
		params.qr = qr
		params.qrMessageNametag = qrMessageNametag
		return nil
	}
}

func WithResponderParameters(applicationName, applicationVersion, shardID string, qrMessageNameTag *n.MessageNametag) PairingParameterOption {
	return func(params *PairingParameters) error {
		params.initiator = false
		if qrMessageNameTag == nil {
			b := make([]byte, n.MessageNametagLength)
			_, err := rand.Read(b)
			if err != nil {
				return err
			}
			params.qrMessageNametag = n.BytesToMessageNametag(b)
		} else {
			params.qrMessageNametag = *qrMessageNameTag
		}
		params.qr = NewQR(applicationName, applicationVersion, shardID, params.ephemeralPublicKey, params.myCommitedStaticKey)
		return nil
	}
}

const DefaultApplicationName = "waku-noise-sessions"
const DefaultApplicationVersion = "0.1"
const DefaultShardID = "10"

func WithDefaultResponderParameters() PairingParameterOption {
	return WithResponderParameters(DefaultApplicationName, DefaultApplicationVersion, DefaultShardID, nil)
}

type PairingParameters struct {
	myCommitedStaticKey []byte
	ephemeralPublicKey  ed25519.PublicKey
	initiator           bool
	qr                  QR
	qrMessageNametag    n.MessageNametag
}

func NewPairing(myStaticKey n.Keypair, myEphemeralKey n.Keypair, opts PairingParameterOption, messenger NoiseMessenger, logger *zap.Logger) (*Pairing, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	myCommitedStaticKey := n.CommitPublicKey(sha256.New, myStaticKey.Public, b)

	var params PairingParameters
	params.myCommitedStaticKey = myCommitedStaticKey
	params.ephemeralPublicKey = myEphemeralKey.Public
	err = opts(&params)
	if err != nil {
		return nil, err
	}

	hs, err := n.NewHandshake_WakuPairing_25519_ChaChaPoly_SHA256(myStaticKey, myEphemeralKey, params.initiator, params.qr.Bytes(), params.qr.ephemeralPublicKey)
	if err != nil {
		return nil, err
	}

	contentTopic := "/" + params.qr.applicationName + "/" + params.qr.applicationVersion + "/wakunoise/1/sessions_shard-" + params.qr.shardID + "/proto"

	// TODO: check if subscription is removed on stop
	msgCh := messenger.Subscribe(context.Background(), contentTopic)

	return &Pairing{
		ContentTopic:         contentTopic,
		msgCh:                msgCh,
		randomFixLenVal:      b, // this represents r or s depending if you're responder or initiator
		myCommittedStaticKey: myCommitedStaticKey,
		authCodeEmitted:      make(chan string, 1),
		authCodeConfirmed:    make(chan bool, 1),
		params:               params,
		handshake:            hs,
		messenger:            messenger,
		logger:               logger.Named("waku-pairing1"),
	}, nil
}

func (p *Pairing) PairingInfo() (qrString string, qrMessageNametag n.MessageNametag) {
	p.RLock()
	defer p.RUnlock()
	return p.params.qr.String(), p.params.qrMessageNametag
}

func (p *Pairing) Execute(ctx context.Context) error {
	p.RLock()
	if p.started {
		p.RUnlock()
		return errors.New("pairing already executed. Create new pairing object")
	}

	defer p.messenger.Stop()

	p.RUnlock()
	p.Lock()
	p.started = true
	p.Unlock()

	var doneCh <-chan error
	if p.params.initiator {
		doneCh = p.initiatorHandshake(ctx, p.msgCh)
	} else {
		doneCh = p.responderHandshake(ctx, p.msgCh)
	}

	select {
	case <-ctx.Done():
		p.Lock()
		defer p.Unlock()
		return ErrPairingTimeout
	case err := <-doneCh:
		return err
	}
}

func (p *Pairing) isAuthCodeConfirmed(ctx context.Context) (bool, error) {
	// wait for user to confirm or not, or for the whole pairing process to time out
	select {
	case <-ctx.Done():
		return false, ErrPairingTimeout
	case confirmed := <-p.authCodeConfirmed:
		return confirmed, nil
	}
}

func (p *Pairing) executeReadStepWithNextMessage(ctx context.Context, nextMsgChan <-chan *pb.WakuMessage, messageNametag n.MessageNametag) (*n.HandshakeStepResult, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ErrPairingTimeout
		case msg := <-nextMsgChan:
			payload, err := DecodePayloadV2(msg)
			if err != nil {
				return nil, err
			}

			step, err := p.handshake.Step(payload, nil, messageNametag)
			if err != nil {
				if errors.Is(err, n.ErrNametagNotExpected) || errors.Is(err, n.ErrUnexpectedMessageNametag) {
					p.logger.Debug(err.Error())
					continue
				}

				return nil, err
			}
			return step, nil
		}
	}
}

func (p *Pairing) initiatorHandshake(ctx context.Context, msgCh <-chan *pb.WakuMessage) (doneCh chan error) {
	doneCh = make(chan error, 1)

	go func() {
		defer utils.LogOnPanic()
		defer close(doneCh)
		// The handshake initiator writes a Waku2 payload v2 containing the handshake message
		// and the (encrypted) transport message
		// The message is sent with a messageNametag equal to the one received through the QR code
		hsStep, err := p.handshake.Step(nil, p.myCommittedStaticKey, p.params.qrMessageNametag)
		if err != nil {
			doneCh <- err
			return
		}

		// We prepare a message from initiator's payload2
		// At this point wakuMsg is sent over the Waku network to receiver content topic
		err = p.messenger.Publish(ctx, p.ContentTopic, hsStep.PayloadV2)
		if err != nil {
			doneCh <- err
			return
		}
		// We generate an authorization code using the handshake state
		// this check has to be confirmed with a user interaction, comparing auth codes in both ends
		authCode, err := p.handshake.Authcode()
		if err != nil {
			doneCh <- err
			return
		}

		p.Lock()
		p.authCode = authCode
		p.Unlock()

		p.authCodeEmitted <- authCode

		p.logger.Info("waiting for authcode confirmation....")

		confirmed, err := p.isAuthCodeConfirmed(ctx)
		if err != nil {
			doneCh <- err
			return
		}
		if !confirmed {
			p.logger.Info("authcode not confirmed")
			doneCh <- errors.New("authcode not confirmed")
			return
		}

		// 2nd step
		// <- sB, eAsB    {r}
		hsMessageNametag, err := p.handshake.ToMessageNametag()
		if err != nil {
			doneCh <- err
			return
		}

		hsStep, err = p.executeReadStepWithNextMessage(ctx, msgCh, hsMessageNametag)
		if err != nil {
			doneCh <- err
			return
		}

		// Initiator further checks if receiver's commitment opens to receiver's static key received
		expectedReceiverCommittedStaticKey := n.CommitPublicKey(sha256.New, p.handshake.RemoteStaticPublicKey(), hsStep.TransportMessage)
		if !bytes.Equal(expectedReceiverCommittedStaticKey, p.params.qr.committedStaticKey) {
			doneCh <- errors.New("expected committed static key does not match the receiver actual committed static key")
			return
		}

		// 3rd step
		// -> sA, sAeB, sAsB  {s}
		// Similarly as in first step, the initiator writes a Waku2 payload containing the handshake message and the (encrypted) transport message
		hsMessageNametag, err = p.handshake.ToMessageNametag()
		if err != nil {
			doneCh <- err
			return
		}

		hsStep, err = p.handshake.Step(nil, p.randomFixLenVal, hsMessageNametag)
		if err != nil {
			doneCh <- err
			return
		}

		err = p.messenger.Publish(ctx, p.ContentTopic, hsStep.PayloadV2)
		if err != nil {
			doneCh <- err
			return
		}

		hsResult, err := p.handshake.FinalizeHandshake()
		if err != nil {
			doneCh <- err
			return
		}

		// Secure Transfer Phase
		if !p.handshake.IsComplete() {
			doneCh <- errors.New("handshake is in undefined state")
			return
		}

		p.Lock()
		p.hsResult = hsResult
		p.completed = true
		p.Unlock()
	}()

	return doneCh
}

func (p *Pairing) responderHandshake(ctx context.Context, msgCh <-chan *pb.WakuMessage) (doneCh chan error) {
	doneCh = make(chan error, 1)

	func() {
		defer close(doneCh)

		// the received reads the initiator's payloads, and returns the (decrypted) transport message the initiator sent
		// Note that the received verifies if the received payloadV2 has the expected messageNametag set
		hsStep, err := p.executeReadStepWithNextMessage(ctx, msgCh, p.params.qrMessageNametag)
		if err != nil {
			doneCh <- err
			return
		}

		initiatorCommittedStaticKey := hsStep.TransportMessage

		// We generate an authorization code using the handshake state
		// this check has to be confirmed with a user interaction, comparing auth codes in both ends
		authCode, err := p.handshake.Authcode()
		if err != nil {
			doneCh <- err
			return
		}

		p.Lock()
		p.authCode = authCode
		p.Unlock()

		p.authCodeEmitted <- authCode

		p.logger.Info("waiting for authcode confirmation....")

		confirmed, err := p.isAuthCodeConfirmed(ctx)
		if err != nil {
			doneCh <- err
			return
		}
		if !confirmed {
			p.logger.Info("authcode not confirmed")
			doneCh <- errors.New("authcode not confirmed")
			return
		}

		// 2nd step
		// <- sB, eAsB    {r}
		// Receiver writes and returns a payload
		hsMessageNametag, err := p.handshake.ToMessageNametag()
		if err != nil {
			doneCh <- err
			return
		}

		hsStep, err = p.handshake.Step(nil, p.randomFixLenVal, hsMessageNametag)
		if err != nil {
			doneCh <- err
			return
		}

		// We prepare a Waku message from receiver's payload2
		err = p.messenger.Publish(ctx, p.ContentTopic, hsStep.PayloadV2)
		if err != nil {
			doneCh <- err
			return
		}

		// 3rd step
		// -> sA, sAeB, sAsB  {s}
		hsMessageNametag, err = p.handshake.ToMessageNametag()
		if err != nil {
			doneCh <- err
			return
		}

		// The receiver reads the initiator's payload sent by the initiator
		hsStep, err = p.executeReadStepWithNextMessage(ctx, msgCh, hsMessageNametag)
		if err != nil {
			doneCh <- err
			return
		}

		// The receiver further checks if the initiator's commitment opens to the initiator's static key received
		expectedInitiatorCommittedStaticKey := n.CommitPublicKey(sha256.New, p.handshake.RemoteStaticPublicKey(), hsStep.TransportMessage)
		if !bytes.Equal(expectedInitiatorCommittedStaticKey, initiatorCommittedStaticKey) {
			doneCh <- errors.New("expected committed static key does not match the initiator actual committed static key")
			return
		}

		hsResult, err := p.handshake.FinalizeHandshake()
		if err != nil {
			doneCh <- err
			return
		}

		// Secure Transfer Phase
		if !p.handshake.IsComplete() {
			doneCh <- errors.New("handshake is in undefined state")
			return
		}

		p.Lock()
		p.completed = true
		p.hsResult = hsResult
		p.Unlock()
	}()
	return doneCh
}

func (p *Pairing) HandshakeComplete() bool {
	p.RLock()
	defer p.RUnlock()
	return p.completed
}

// Returns a WakuMessage with version 2 and encrypted payload
func (p *Pairing) Encrypt(plaintext []byte) (*pb.WakuMessage, error) {
	p.RLock()
	defer p.RUnlock()
	if !p.completed {
		return nil, errors.New("pairing is not complete")
	}

	payload, err := p.hsResult.WriteMessage(plaintext, nil)
	if err != nil {
		return nil, err
	}

	encodedPayload, err := EncodePayloadV2(payload)
	if err != nil {
		return nil, err
	}

	encodedPayload.ContentTopic = p.ContentTopic

	return encodedPayload, nil
}

func (p *Pairing) Decrypt(msg *pb.WakuMessage) ([]byte, error) {
	p.RLock()
	defer p.RUnlock()
	if !p.completed {
		return nil, errors.New("pairing is not complete")
	}

	payload, err := DecodePayloadV2(msg)
	if err != nil {
		return nil, err
	}
	return p.hsResult.ReadMessage(payload, nil)
}

func (p *Pairing) ConfirmAuthCode(confirmed bool) error {
	p.RLock()
	authcode := p.authCode
	p.RUnlock()

	if authcode != "" {
		p.authCodeConfirmed <- confirmed
		return nil
	}

	return errors.New("authcode has not been generated yet")
}

func (p *Pairing) AuthCode() <-chan string {
	ch := make(chan string, 1)

	p.Lock()
	authcode := p.authCode
	p.Unlock()

	if authcode != "" {
		ch <- authcode
	} else {
		ch <- <-p.authCodeEmitted
	}
	close(ch)
	return ch
}
