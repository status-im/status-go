package reliability

import (
	"bytes"
	"crypto/ecdsa"
	"database/sql"
	"errors"

	mvdsproto "github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"
	"github.com/status-im/mvds/store"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability/datasync"
	datasyncpeer "github.com/status-im/status-go/pkg/messaging/layers/reliability/datasync/peer"
)

// Uncompressed public-key peer IDs start with 4. Installation IDs belong to a
// separate namespace in the same MVDS engine and SQLite stores.
const installationPeerPrefix = 1

type InstallationDispatcher func(state.PeerID, []byte, [][]byte) error

func InstallationPeerID(profile *ecdsa.PublicKey, installation string) state.PeerID {
	var peer state.PeerID
	peer[0] = installationPeerPrefix
	copy(peer[1:], crypto.Keccak256(crypto.CompressPubkey(profile), []byte(installation)))
	return peer
}

func installationGroup(profile *ecdsa.PublicKey, first, second string) state.GroupID {
	if first > second {
		first, second = second, first
	}
	key := crypto.CompressPubkey(profile)
	firstPeer := InstallationPeerID(profile, first)
	secondPeer := InstallationPeerID(profile, second)
	return datasync.ToGroupID(crypto.Keccak256([]byte("status-installation-mvds"), key, firstPeer[:], secondPeer[:]))
}

func (r *Reliability) installationPeer(ds *datasync.DataSync, local, remote string) (state.GroupID, state.PeerID, error) {
	group := installationGroup(&r.identity.PublicKey, local, remote)
	peer := InstallationPeerID(&r.identity.PublicKey, remote)
	exists, err := ds.IsPeerInGroup(group, peer)
	if err == nil && !exists {
		err = ds.AddPeer(group, peer)
	}
	return group, peer, err
}

func (r *Reliability) QueueInstallationMessage(local, remote string, message []byte) (state.MessageID, error) {
	group := installationGroup(&r.identity.PublicKey, local, remote)
	return r.queueSelfMessage(group, InstallationPeerID(&r.identity.PublicKey, remote), message)
}

func (r *Reliability) QueueLegacySelfMessage(message []byte) (state.MessageID, error) {
	key := &r.identity.PublicKey
	return r.queueSelfMessage(datasync.ToOneToOneGroupID(key, key), datasyncpeer.PublicKeyToPeerID(*key), message)
}

func (r *Reliability) queueSelfMessage(group state.GroupID, peer state.PeerID, message []byte) (state.MessageID, error) {
	ds := r.datasync.Load()
	if ds == nil {
		// The persistent queue is available while the outbound loop is offline.
		var err error
		ds, err = r.newDataSync(datasync.DatasyncTicker)
		if err != nil {
			return state.MessageID{}, err
		}
		defer ds.Stop()
	}
	return queueMessage(ds, group, peer, message)
}

// ProcessInstallationPayload routes standard MVDS data and ACKs independently.
// A direct encryption session identifies the installation, while the group
// distinguishes a delivery queued by installation from a profile-wide send.
func (r *Reliability) ProcessInstallationPayload(local, remote string, encoded []byte, enabled bool) (scoped, legacy *mvdsproto.Payload, err error) {
	ds := r.datasync.Load()
	if ds == nil {
		return nil, nil, errNotStarted
	}
	payload, err := datasync.DecodePayload(encoded)
	if err != nil {
		return nil, nil, err
	}
	if !payload.IsValid() {
		return nil, nil, errors.New("invalid MVDS payload")
	}

	key := &r.identity.PublicKey
	scopedGroup := installationGroup(key, local, remote)
	legacyGroup := datasync.ToOneToOneGroupID(key, key)
	scoped = &mvdsproto.Payload{}
	legacy = &mvdsproto.Payload{
		Offers: payload.Offers, Requests: payload.Requests, GroupOffers: payload.GroupOffers,
	}
	for _, message := range payload.Messages {
		if message == nil {
			continue
		}
		switch {
		case bytes.Equal(message.GroupId, scopedGroup[:]):
			scoped.Messages = append(scoped.Messages, message)
		case bytes.Equal(message.GroupId, legacyGroup[:]):
			legacy.Messages = append(legacy.Messages, message)
		}
	}

	// An ACK carries only an MVDS message ID. Look up its group so an
	// upgraded installation can still ACK pending profile-wide deliveries.
	messages := r.mvdsPersistence.MessageStore()
	for _, ack := range payload.Acks {
		var id state.MessageID
		if len(ack) != len(id) {
			continue
		}
		copy(id[:], ack)
		message, err := messages.Get(id)
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, store.ErrMessageNotFound) {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		switch {
		case enabled && bytes.Equal(message.GroupId, scopedGroup[:]):
			scoped.Acks = append(scoped.Acks, ack)
		case bytes.Equal(message.GroupId, legacyGroup[:]):
			legacy.Acks = append(legacy.Acks, ack)
		}
	}

	if scoped.IsValid() {
		_, peer, err := r.installationPeer(ds, local, remote)
		if err != nil {
			return nil, nil, err
		}
		if _, err = ds.ProcessPayload(peer, scoped); err != nil {
			return nil, nil, err
		}
	}
	if legacy.IsValid() {
		if _, err = ds.ProcessPayload(datasyncpeer.PublicKeyToPeerID(*key), legacy); err != nil {
			return nil, nil, err
		}
	}
	return scoped, legacy, nil
}

// DropInstallationMessages removes pending retries when a device is disabled
// or deleted. Dispatch also checks that the installation remains enabled.
func (r *Reliability) DropInstallationMessages(installation string) {
	peer := InstallationPeerID(&r.identity.PublicKey, installation)
	store := r.mvdsPersistence.StateStore()
	for {
		var pending []state.MessageID
		err := store.MapWithPeerId(peer, func(item state.State) state.State {
			pending = append(pending, item.MessageID)
			return item
		})
		if err != nil {
			r.logger.Warn("failed to list installation retries", zap.Error(err))
			return
		}
		if len(pending) == 0 {
			return
		}
		for _, id := range pending {
			if err := store.Remove(id, peer); err != nil && !errors.Is(err, state.ErrStateNotFound) {
				r.logger.Warn("failed to remove installation retry", zap.Error(err))
				return
			}
		}
	}
}
