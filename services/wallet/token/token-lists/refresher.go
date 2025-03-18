package tokenlists

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/settings"
)

func (t *TokenLists) startAutoRefreshLoop(ctx context.Context, autoRefreshInterval time.Duration, autoRefreshCheckInterval time.Duration) {
	if t.settings == nil {
		logutils.ZapLogger().Error("settings is nil")
		return
	}
	ticker := time.NewTicker(autoRefreshCheckInterval)
	go func() {
		defer common.LogOnPanic()
		for {
			select {
			case <-ticker.C:
				autoRefreshEnabled, err := t.settings.AutoRefreshTokensEnabled()
				if err != nil {
					logutils.ZapLogger().Error("failed to get auto refresh setting", zap.Error(err))
					continue
				}
				if !autoRefreshEnabled {
					continue
				}
				lastTokensUpdate, err := t.settings.LastTokensUpdate()
				if err != nil {
					logutils.ZapLogger().Error("failed to get last tokens update time", zap.Error(err))
					continue
				}
				if time.Since(lastTokensUpdate) < autoRefreshInterval {
					continue
				}

				storedListsCount, err := t.tokenListsFetcher.FetchAndStore(ctx)
				if err != nil {
					logutils.ZapLogger().Error("failed to fetch and store token lists", zap.Error(err))
					// Just log an error and don't continue, cause we have to store last tokens update timestamp
				}

				currentTimestamp := time.Unix(time.Now().Unix(), 0)
				err = t.settings.SaveSettingField(settings.LastTokensUpdate, currentTimestamp)
				if err != nil {
					logutils.ZapLogger().Error("failed to save last tokens update time", zap.Error(err))
					continue
				}

				if storedListsCount > 0 {
					t.notifyCh <- struct{}{}
				}

			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
