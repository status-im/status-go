package protocol

import (
	"database/sql"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/status-im/status-go/appdatabase/migrations"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/pushnotificationclient"
	"github.com/status-im/status-go/protocol/pushnotificationserver"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/services/mailservers"
)

type MessageDeliveredHandler func(string, string)

type MessengerSignalsHandler interface {
	MessageDelivered(chatID string, messageID string)
	CommunityInfoFound(community *communities.Community)
	MessengerResponse(response *MessengerResponse)
}

type config struct {
	// This needs to be exposed until we move here mailserver logic
	// as otherwise the client is not notified of a new filter and
	// won't be pulling messages from mailservers until it reloads the chats/filters
	onContactENSVerified func(*MessengerResponse)

	// systemMessagesTranslations holds translations for system-messages
	systemMessagesTranslations *systemMessageTranslationsMap
	// Config for the envelopes monitor
	envelopesMonitorConfig *transport.EnvelopesMonitorConfig

	featureFlags common.FeatureFlags

	// A path to a database or a database instance is required.
	// The database instance has a higher priority.
	dbConfig            dbConfig
	db                  *sql.DB
	afterDbCreatedHooks []Option
	multiAccount        *multiaccounts.Database
	mailserversDatabase *mailservers.Database
	account             *multiaccounts.Account

	verifyTransactionClient  EthClient
	verifyENSURL             string
	verifyENSContractAddress string

	pushNotificationServerConfig *pushnotificationserver.Config
	pushNotificationClientConfig *pushnotificationclient.Config

	logger *zap.Logger

	messengerSignalsHandler MessengerSignalsHandler
}

type Option func(*config) error

// WithSystemMessagesTranslations is required for Group Chats which are currently disabled.
// nolint: unused
func WithSystemMessagesTranslations(t map[protobuf.MembershipUpdateEvent_EventType]string) Option {
	return func(c *config) error {
		c.systemMessagesTranslations.Init(t)
		return nil
	}
}

func WithCustomLogger(logger *zap.Logger) Option {
	return func(c *config) error {
		c.logger = logger
		return nil
	}
}

func WithDatabaseConfig(dbPath, dbKey string) Option {
	return func(c *config) error {
		c.dbConfig = dbConfig{dbPath: dbPath, dbKey: dbKey}
		return nil
	}
}

func WithVerifyTransactionClient(client EthClient) Option {
	return func(c *config) error {
		c.verifyTransactionClient = client
		return nil
	}
}

func WithDatabase(db *sql.DB) Option {
	return func(c *config) error {
		c.db = db
		return nil
	}
}

func WithToplevelDatabaseMigrations() Option {
	return func(c *config) error {
		c.afterDbCreatedHooks = append(c.afterDbCreatedHooks, func(c *config) error {
			return migrations.Migrate(c.db)
		})
		return nil
	}
}

func WithAppSettings(s accounts.Settings, nc params.NodeConfig) Option {
	return func(c *config) error {
		c.afterDbCreatedHooks = append(c.afterDbCreatedHooks, func(c *config) error {
			if s.Networks == nil {
				networks := new(json.RawMessage)
				if err := networks.UnmarshalJSON([]byte("net")); err != nil {
					return err
				}

				s.Networks = networks
			}

			sDB := accounts.NewDB(c.db)
			return sDB.CreateSettings(s, nc)
		})
		return nil
	}
}

func WithMultiAccounts(ma *multiaccounts.Database) Option {
	return func(c *config) error {
		c.multiAccount = ma
		return nil
	}
}

func WithMailserversDatabase(ma *mailservers.Database) Option {
	return func(c *config) error {
		c.mailserversDatabase = ma
		return nil
	}
}

func WithAccount(acc *multiaccounts.Account) Option {
	return func(c *config) error {
		c.account = acc
		return nil
	}
}

func WithPushNotificationServerConfig(pushNotificationServerConfig *pushnotificationserver.Config) Option {
	return func(c *config) error {
		c.pushNotificationServerConfig = pushNotificationServerConfig
		return nil
	}
}

func WithPushNotificationClientConfig(pushNotificationClientConfig *pushnotificationclient.Config) Option {
	return func(c *config) error {
		c.pushNotificationClientConfig = pushNotificationClientConfig
		return nil
	}
}

func WithDatasync() func(c *config) error {
	return func(c *config) error {
		c.featureFlags.Datasync = true
		return nil
	}
}

func WithPushNotifications() func(c *config) error {
	return func(c *config) error {
		c.featureFlags.PushNotifications = true
		return nil
	}
}

func WithEnvelopesMonitorConfig(emc *transport.EnvelopesMonitorConfig) Option {
	return func(c *config) error {
		c.envelopesMonitorConfig = emc
		return nil
	}
}

func WithSignalsHandler(h MessengerSignalsHandler) Option {
	return func(c *config) error {
		c.messengerSignalsHandler = h
		return nil
	}
}

func WithENSVerificationConfig(onENSVerified func(*MessengerResponse), url, address string) Option {
	return func(c *config) error {
		c.onContactENSVerified = onENSVerified
		c.verifyENSURL = url
		c.verifyENSContractAddress = address
		return nil
	}
}
