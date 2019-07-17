package publisher

import (
	"crypto/ecdsa"
	"database/sql"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
)

const (
	// How often a ticker fires in seconds.
	tickerInterval = 120
	// How often we should publish a contact code in seconds.
	publishInterval = 21600
	// Cooldown period on acking messages when not targeting our device.
	deviceNotFoundAckInterval = 7200
)

var (
	errNotEnoughTimePassed = errors.New("not enough time passed")
)

type Publisher struct {
	persistence *sqlitePersistence
	logger      *zap.Logger
	notifyCh    chan struct{}
	quit        chan struct{}
}

func New(db *sql.DB, logger *zap.Logger) *Publisher {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Publisher{
		persistence: newSQLitePersistence(db),
		logger:      logger.With(zap.Namespace("Publisher")),
	}
}

func (p *Publisher) Start() <-chan struct{} {
	logger := p.logger.With(zap.String("site", "Start"))

	logger.Info("starting publisher")

	p.notifyCh = make(chan struct{})
	p.quit = make(chan struct{})

	go p.tickerLoop()

	return p.notifyCh
}

func (p *Publisher) Stop() {
	select {
	case _, ok := <-p.quit:
		if !ok {
			// channel already closed
			return
		}
	default:
		close(p.quit)
	}
}

func (p *Publisher) tickerLoop() {
	ticker := time.NewTicker(tickerInterval * time.Second)

	go func() {
		logger := p.logger.With(zap.String("site", "tickerLoop"))

		for {
			select {
			case <-ticker.C:
				err := p.notify()
				switch err {
				case errNotEnoughTimePassed:
					logger.Debug("not enough time passed")
				case nil:
					// skip
				default:
					logger.Error("error while sending a contact code", zap.Error(err))
				}
			case <-p.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (p *Publisher) notify() error {
	lastPublished, err := p.persistence.lastPublished()
	if err != nil {
		return err
	}

	now := time.Now().Unix()

	if now-lastPublished < publishInterval {
		return errNotEnoughTimePassed
	}

	p.notifyCh <- struct{}{}

	return p.persistence.setLastPublished(now)
}

func (p *Publisher) ShouldAdvertiseBundle(publicKey *ecdsa.PublicKey, now int64) (bool, error) {
	identity := crypto.CompressPubkey(publicKey)
	lastAcked, err := p.persistence.lastAck(identity)
	if err != nil {
		return false, err
	}
	return now-lastAcked < deviceNotFoundAckInterval, nil
}

func (p *Publisher) SetLastAck(publicKey *ecdsa.PublicKey, now int64) {
	identity := crypto.CompressPubkey(publicKey)
	p.persistence.setLastAck(identity, now)
}
