package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/pkg/errors"
	"github.com/status-im/status-console-client/protocol/v1"
)

type Messenger struct {
	identity *ecdsa.PrivateKey
	proto    protocol.Protocol
	db       Database

	mu      sync.Mutex         // guards public and private maps
	public  map[string]*Stream // key is Contact.Topic
	private map[string]*Stream // key is Contact.Topic

	events *event.Feed
}

func NewMessenger(identity *ecdsa.PrivateKey, proto protocol.Protocol, db Database) *Messenger {
	events := &event.Feed{}
	return &Messenger{
		identity: identity,
		proto:    proto,
		db:       NewDatabaseWithEvents(db, events),

		public:  make(map[string]*Stream),
		private: make(map[string]*Stream),
		events:  events,
	}
}

func (m *Messenger) Start() error {
	log.Printf("[Messenger::Start]")

	m.mu.Lock()
	defer m.mu.Unlock()

	contacts, err := m.db.Contacts()
	if err != nil {
		return errors.Wrap(err, "unable to read contacts from database")
	}

	// For each contact, a new Stream is created that subscribes to the protocol
	// and forwards incoming messages to the Stream instance.
	// We iterate over all contacts, however, there are two cases:
	// (1) Public chats where each has a unique topic,
	// (2) Private chats where a single or shared topic is used.
	// This means that we don't know from which contact the message
	// came from until it is examined.
	for i := range contacts {
		if err := m.addStream(contacts[i]); err != nil {
			return err
		}
	}

	log.Printf("[Messenger::Start] request messages from mail sever")

	return m.RequestAll(context.Background(), true)
}

// addStream creates a new Stream and adds it to the Messenger.
// For contacts with public key, we just need to make sure
// each possible topic has a stream. For a single topic
// for all private conversations, the map will have a len of 1.
// In the future, private conversations will have sharded topics,
// which means there will be many conversation over a particular topic
// but there will be more than one topic.
func (m *Messenger) addStream(c Contact) error {
	options, err := createSubscribeOptions(c)
	if err != nil {
		return errors.Wrap(err, "failed to create SubscribeOptions")
	}

	switch c.Type {
	case ContactPrivate:
		_, exist := m.private[c.Topic]
		if exist {
			return nil
		}

		stream := NewStream(
			m.proto,
			StreamStoreHandlerMultiplexed(m.db),
		)
		if err := stream.Start(context.Background(), options); err != nil {
			return errors.Wrap(err, "can't subscribe to a stream")
		}

		m.private[c.Name] = stream
	case ContactPublicRoom:
		_, exist := m.public[c.Topic]
		if exist {
			return nil
		}

		stream := NewStream(
			m.proto,
			StreamStoreHandlerForContact(m.db, c),
		)
		if err := stream.Start(context.Background(), options); err != nil {
			return errors.Wrap(err, "can't subscribe to a stream")
		}

		m.public[c.Name] = stream
	default:
		return fmt.Errorf("unsupported contect type: %s", c.Type)
	}

	return nil
}

func (m *Messenger) Join(ctx context.Context, c Contact) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.addStream(c); err != nil {
		return err
	}

	opts := protocol.DefaultRequestOptions()
	if err := m.Request(ctx, c, opts); err != nil {
		return err
	}
	return m.db.UpdateHistories([]History{{Contact: c, Synced: opts.To}})
}

// Messages reads all messages from database.
func (m *Messenger) Messages(c Contact, offset int64) ([]*protocol.Message, error) {
	return m.db.NewMessages(c, offset)
}

func (m *Messenger) Request(ctx context.Context, c Contact, options protocol.RequestOptions) error {
	err := enhanceRequestOptions(c, &options)
	if err != nil {
		return err
	}
	return m.proto.Request(ctx, options)
}

func (m *Messenger) requestHistories(ctx context.Context, histories []History, opts protocol.RequestOptions) error {
	log.Printf("[Messenger::requestHistories] requesting messages for chats %+v: from %d to %d\n", opts.Chats, opts.From, opts.To)

	start := time.Now()

	err := m.proto.Request(ctx, opts)
	if err != nil {
		return err
	}

	log.Printf("[Messenger::requestHistories] requesting message for chats %+v finished in %s\n", opts.Chats, time.Since(start))

	for i := range histories {
		histories[i].Synced = opts.To
	}
	return m.db.UpdateHistories(histories)
}

func (m *Messenger) RequestAll(ctx context.Context, newest bool) error {
	// FIXME(dshulyak) if newest is false request 24 hour of messages older then the
	// earliest envelope for each contact.
	histories, err := m.db.Histories()
	if err != nil {
		return errors.Wrap(err, "error fetching contacts")
	}
	var (
		now               = time.Now()
		synced, notsynced = splitIntoSyncedNotSynced(histories)
		errors            = make(chan error, 2)
		wg                sync.WaitGroup
	)
	if len(synced) != 0 {
		wg.Add(1)
		go func() {
			errors <- m.requestHistories(ctx, synced, syncedToOpts(synced, now))
			wg.Done()
		}()
	}
	if len(notsynced) != 0 {
		wg.Add(1)
		go func() {
			errors <- m.requestHistories(ctx, notsynced, notsyncedToOpts(notsynced, now))
			wg.Done()
		}()
	}
	wg.Wait()

	log.Printf("[Messenger::RequestAll] finished requesting histories")

	close(errors)
	for err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Messenger) Send(c Contact, data []byte) error {
	// FIXME(dshulyak) sending must be locked by contact to prevent sending second msg with same clock
	clock, err := m.db.LastMessageClock(c)
	if err != nil {
		return errors.Wrap(err, "failed to read last message clock for contact")
	}
	var message protocol.Message

	switch c.Type {
	case ContactPublicRoom:
		message = protocol.CreatePublicTextMessage(data, clock, c.Name)
	case ContactPrivate:
		message = protocol.CreatePrivateTextMessage(data, clock, c.Name)
	default:
		return fmt.Errorf("failed to send message: unsupported contact type")
	}

	encodedMessage, err := protocol.EncodeMessage(message)
	if err != nil {
		return errors.Wrap(err, "failed to encode message")
	}
	opts, err := createSendOptions(c)
	if err != nil {
		return errors.Wrap(err, "failed to prepare send options")
	}

	log.Printf("[Messenger::Send] sending message")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	hash, err := m.proto.Send(ctx, encodedMessage, opts)
	if err != nil {
		return errors.Wrap(err, "can't send a message")
	}

	log.Printf("[Messenger::Send] sent message with hash %x", hash)

	message.ID = hash
	message.SigPubKey = &m.identity.PublicKey
	_, err = m.db.SaveMessages(c, []*protocol.Message{&message})
	if err != nil {
		return errors.Wrap(err, "failed to save the message")
	}
	return nil
}

func (m *Messenger) RemoveContact(c Contact) error {
	return m.db.DeleteContact(c)
}

func (m *Messenger) AddContact(c Contact) error {
	return m.db.SaveContacts([]Contact{c})
}

func (m *Messenger) Contacts() ([]Contact, error) {
	return m.db.Contacts()
}

func (m *Messenger) Leave(c Contact) error {
	var stream *Stream

	m.mu.Lock()
	defer m.mu.Unlock()

	switch c.Type {
	case ContactPublicRoom:
		stream = m.public[c.Topic]
		if stream != nil {
			delete(m.public, c.Topic)
		}
	case ContactPrivate:
		// TODO: should we additionally block that peer?
		stream = m.private[c.Topic]
		if stream != nil {
			delete(m.private, c.Topic)
		}
	}

	if stream == nil {
		return errors.New("stream not found")
	}

	stream.Stop()

	return nil
}

func (m *Messenger) Subscribe(events chan Event) event.Subscription {
	return m.events.Subscribe(events)
}
