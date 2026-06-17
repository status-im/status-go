package protocol

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	utils "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/pinnedcommunities"
	"github.com/status-im/status-go/protocol/protobuf"
)

func (m *Messenger) bootstrapPinnedCommunities() error {
	shouldBootstrap, err := m.shouldBootstrapPinnedCommunities()
	if err != nil {
		return err
	}
	if !shouldBootstrap {
		m.logger.Debug("skipping pinned communities bootstrap: profile already has communities")
		return nil
	}

	dir := strings.TrimSpace(os.Getenv("STATUS_GO_PINNED_COMMUNITIES_DIR"))
	source := "embedded"

	var payloads []pinnedcommunities.Payload
	if dir != "" {
		source = "dir"
		payloads, err = pinnedcommunities.LoadFromDir(dir)
		if err != nil {
			if os.IsNotExist(err) || errors.Is(err, fs.ErrNotExist) {
				m.logger.Debug("pinned communities dir does not exist", zap.String("dir", dir))
				return nil
			}
			return err
		}
	} else {
		payloads, err = pinnedcommunities.LoadEmbedded()
		if err != nil {
			return err
		}
	}

	if len(payloads) == 0 {
		if source == "dir" {
			m.logger.Debug("no pinned community payloads found", zap.String("dir", dir))
		} else {
			m.logger.Debug("no embedded pinned community payloads found")
		}
		return nil
	}

	if source == "dir" {
		m.logger.Info("bootstrapping pinned communities",
			zap.Int("count", len(payloads)),
			zap.String("source", source),
			zap.String("dir", dir),
		)
	} else {
		m.logger.Info("bootstrapping pinned communities",
			zap.Int("count", len(payloads)),
			zap.String("source", source),
		)
	}

	response := &MessengerResponse{}

	for _, p := range payloads {
		if m.shouldAbortPinnedCommunitiesBootstrap() {
			m.logger.Debug("aborting pinned communities bootstrap: shutting down")
			return nil
		}

		importedCommunity, err := m.importPinnedCommunityPayload(p)
		if err != nil {
			m.logger.Warn("failed to import pinned community payload",
				zap.String("communityID", p.CommunityID),
				zap.String("file", p.FileName),
				zap.Error(err),
			)
			continue
		}

		if importedCommunity != nil {
			response.AddCommunity(importedCommunity)
		}

		if m.shouldAbortPinnedCommunitiesBootstrap() {
			m.logger.Debug("aborting pinned communities bootstrap before refresh: shutting down")
			return nil
		}

		// Force a network refresh so pinned metadata is quickly replaced by fresher store-node data.
		_, _, err = m.storeNodeRequestsManager.FetchCommunity(m.ctx, p.CommunityID, []StoreNodeRequestOption{WithWaitForResponseOption(false)})
		if err != nil {
			m.logger.Warn("failed to refresh pinned community from store node",
				zap.String("communityID", p.CommunityID),
				zap.Error(err),
			)
		}
	}

	if m.shouldAbortPinnedCommunitiesBootstrap() {
		m.logger.Debug("aborting pinned communities bootstrap before publishing response: shutting down")
		return nil
	}

	m.PublishMessengerResponse(response)

	return nil
}

func (m *Messenger) shouldAbortPinnedCommunitiesBootstrap() bool {
	select {
	case <-m.quit:
		return true
	case <-m.ctx.Done():
		return true
	default:
		return false
	}
}

func (m *Messenger) shouldBootstrapPinnedCommunities() (bool, error) {
	count, err := m.communitiesManager.Count()
	if err != nil {
		return false, err
	}

	return count == 0, nil
}

func (m *Messenger) importPinnedCommunityPayload(payload pinnedcommunities.Payload) (*communities.Community, error) {
	var metadata protobuf.ApplicationMetadataMessage
	if err := proto.Unmarshal(payload.RawPayload, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal application metadata: %w", err)
	}

	if metadata.Type != protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION {
		return nil, fmt.Errorf("unsupported metadata type %s", metadata.Type.String())
	}

	signer, err := utils.RecoverKey(&metadata)
	if err != nil {
		return nil, fmt.Errorf("recover signer: %w", err)
	}
	if signer == nil {
		return nil, fmt.Errorf("recover signer: nil signer")
	}

	var description protobuf.CommunityDescription
	if err := proto.Unmarshal(metadata.Payload, &description); err != nil {
		return nil, fmt.Errorf("unmarshal community description: %w", err)
	}

	response, err := m.communitiesManager.HandleCommunityDescriptionMessage(signer, &description, payload.RawPayload, signer)
	if err != nil {
		if err == communities.ErrInvalidCommunityDescriptionClockOutdated {
			return nil, nil
		}
		return nil, fmt.Errorf("handle community description: %w", err)
	}

	if response == nil || response.Community == nil {
		return nil, nil
	}

	if !strings.EqualFold(response.Community.IDString(), payload.CommunityID) {
		payloadHash := sha256.Sum256(payload.RawPayload)
		m.logger.Warn("pinned community id mismatch",
			zap.String("filenameCommunityID", payload.CommunityID),
			zap.String("payloadCommunityID", response.Community.IDString()),
			zap.String("payloadSHA256", fmt.Sprintf("%x", payloadHash[:])),
		)
	}

	return response.Community, nil
}
