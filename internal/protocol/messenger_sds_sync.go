package protocol

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
)

// Match nim-sds's default sync interval.
const sdsSyncInterval = 30 * time.Second

func (m *Messenger) startSDSPeriodicSync() {
	m.shutdownWaitGroup.Add(1)
	go func() {
		defer panics.LogOnPanic()
		defer m.shutdownWaitGroup.Done()
		timer := time.NewTimer(0)
		defer timer.Stop()
		for {
			select {
			case <-m.ctx.Done():
				return
			case <-timer.C:
				active := m.messaging.SDSPeriodicSyncContext()
				if active != nil && m.messaging.Online() {
					ctx, cancel := context.WithCancel(m.ctx)
					stop := context.AfterFunc(active, cancel)
					m.sendSDSPeriodicSyncs(ctx)
					stop()
					cancel()
				}
				timer.Reset(sdsSyncInterval)
			}
		}
	}()
}

func (m *Messenger) sendSDSPeriodicSyncs(ctx context.Context) {
	communities, err := m.communitiesManager.Joined()
	if err != nil {
		m.logger.Error("failed to enumerate communities for SDS sync", zap.Error(err))
		return
	}

	type syncRoute struct {
		communityID string
		pubsubTopic string
		syncTopic   string
	}
	routes := make(map[string]syncRoute, len(communities))
	for _, community := range communities {
		if ctx.Err() != nil {
			return
		}
		channelID := reliability.BuildChannelID(community.IDString(), community.UniversalChatID())
		if err := m.messaging.EnsureSDSChannel(channelID); err != nil {
			m.logger.Warn("failed to ensure SDS channel for periodic sync",
				zap.String("communityID", community.IDString()), zap.Error(err))
			continue
		}
		routes[channelID] = syncRoute{
			communityID: community.IDString(),
			pubsubTopic: community.PubsubTopic(),
			syncTopic:   communitySDSSyncTopic(community.IDString()),
		}
	}

	if ctx.Err() != nil {
		return
	}
	events, err := m.messaging.TickSDS()
	if err != nil {
		m.logger.Error("failed to tick SDS periodic sync", zap.Error(err))
		return
	}
	for _, event := range events {
		if ctx.Err() != nil {
			return
		}
		route, ok := routes[event.ChannelID]
		if !ok {
			m.logger.Debug("skipping SDS sync for inactive community channel",
				zap.String("channelID", event.ChannelID))
			continue
		}
		if event.Error != "" {
			m.logger.Warn("failed to generate SDS sync",
				zap.String("communityID", route.communityID),
				zap.String("channelID", event.ChannelID),
				zap.String("error", event.Error))
			continue
		}
		if len(event.Payload) == 0 {
			m.logger.Warn("SDS tick returned an empty sync payload",
				zap.String("communityID", route.communityID),
				zap.String("channelID", event.ChannelID))
			continue
		}
		// All community chats currently share this history, including restricted
		// chats. The public sync exposes its metadata, but not message content.
		err := m.messaging.SendSDSSync(ctx, event.Payload, route.pubsubTopic, route.syncTopic)
		if err != nil && ctx.Err() == nil {
			m.logger.Warn("failed to publish SDS sync",
				zap.String("communityID", route.communityID), zap.Error(err))
		}
	}
}
