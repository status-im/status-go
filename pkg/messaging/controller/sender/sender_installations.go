package sender

import (
	"github.com/golang/protobuf/proto"
	"github.com/status-im/mvds/state"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/pkg/messaging/common"
	"github.com/status-im/status-go/pkg/messaging/layers/encryption"
	"github.com/status-im/status-go/pkg/messaging/layers/encryption/multidevice"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability/datasync"
	"github.com/status-im/status-go/pkg/messaging/types"
)

func (s *Sender) scheduleInstallationSend(params types.SendPrivateParams, messageID []byte) error {
	s.installationSendMu.Lock()
	defer s.installationSendMu.Unlock()

	devices := s.stack.Encryption.GetMultiDevice()
	installations, err := devices.GetOurInstallations(&s.identity.PublicKey)
	if err != nil {
		return err
	}
	local := devices.InstallationID()
	useLegacy := false
	for _, installation := range installations {
		if installation.ID == local || !installation.Enabled {
			continue
		}
		if installation.Version < encryption.InstallationMVDSVersion {
			useLegacy = true
			continue
		}
		id, err := s.stack.Reliability.QueueInstallationMessage(local, installation.ID, params.Payload)
		if err != nil {
			return err
		}
		if params.TrackSelfDelivery {
			peer := reliability.InstallationPeerID(&s.identity.PublicKey, installation.ID)
			err = s.confirmations.InsertPendingConfirmation(&common.MessageConfirmation{
				MessageID: messageID, DataSyncID: id[:], PublicKey: peer[:],
			})
			if err != nil {
				return err
			}
		}
	}

	if useLegacy {
		id, err := s.stack.Reliability.QueueLegacySelfMessage(params.Payload)
		if err != nil {
			return err
		}
		if params.TrackSelfDelivery {
			return s.confirmations.InsertPendingConfirmation(&common.MessageConfirmation{
				MessageID: messageID, DataSyncID: id[:], PublicKey: crypto.CompressPubkey(params.Recipient),
			})
		}
	}
	return nil
}

func (s *Sender) SendInstallationReliability(peer state.PeerID, payload []byte, messages [][]byte) error {
	s.installationSendMu.Lock()
	defer s.installationSendMu.Unlock()

	devices := s.stack.Encryption.GetMultiDevice()
	local := devices.InstallationID()
	installations, err := devices.GetOurInstallations(&s.identity.PublicKey)
	if err != nil {
		return err
	}
	for _, installation := range installations {
		if installation.ID == local ||
			installation.Version < encryption.InstallationMVDSVersion ||
			reliability.InstallationPeerID(&s.identity.PublicKey, installation.ID) != peer {
			continue
		}
		if !installation.Enabled {
			// Legacy pairing can be one-way. Reply to a known installation's
			// inbound data, while only enabled devices receive outbound data.
			packet, err := datasync.DecodePayload(payload)
			if err != nil || len(packet.Acks) == 0 {
				return err
			}
			packet.Messages = nil
			packet.Offers = nil
			packet.Requests = nil
			payload, err = proto.Marshal(&packet)
			if err != nil {
				return err
			}
			messages = nil
		}
		spec, err := s.stack.Encryption.BuildInstallationMessage(s.identity, []*multidevice.Installation{installation}, payload)
		if err != nil {
			return err
		}
		// The target may have missed the negotiation that installed our
		// shared topic. Use the profile's partitioned topic with per-device encryption.
		spec.AgreedSecret = false
		return s.sendReliableEnvelope(&s.identity.PublicKey, spec, messages)
	}
	return nil
}

func (s *Sender) sendLegacySelfMessages(payload []byte, messages [][]byte) error {
	devices := s.stack.Encryption.GetMultiDevice()
	installations, err := devices.GetOurInstallations(&s.identity.PublicKey)
	if err != nil {
		return err
	}
	local := devices.InstallationID()
	var enabled, legacy []*multidevice.Installation
	for _, installation := range installations {
		if !installation.Enabled || installation.ID == local {
			continue
		}
		enabled = append(enabled, installation)
		if installation.Version < encryption.InstallationMVDSVersion {
			legacy = append(legacy, installation)
		}
	}

	// If all targets have upgraded, finish delivering entries already in the
	// legacy queue using their original wire format and ID.
	targets := legacy
	if len(targets) == 0 {
		targets = enabled
	}
	if len(targets) == 0 {
		return nil
	}
	spec, err := s.stack.Encryption.BuildInstallationMessage(s.identity, targets, payload)
	if err != nil {
		return err
	}
	spec.AgreedSecret = false
	return s.sendReliableEnvelope(&s.identity.PublicKey, spec, messages)
}

func (s *Sender) DropInstallationMessages(installation string) {
	s.installationSendMu.Lock()
	defer s.installationSendMu.Unlock()
	s.stack.Reliability.DropInstallationMessages(installation)
}
