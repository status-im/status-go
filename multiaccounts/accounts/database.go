package accounts

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/sqlite"
)

const (
	uniqueChatConstraint   = "UNIQUE constraint failed: accounts.chat"
	uniqueWalletConstraint = "UNIQUE constraint failed: accounts.wallet"
)

const (
	ProfilePicturesVisibilityContactsOnly = iota + 1
	ProfilePicturesVisibilityEveryone
	ProfilePicturesVisibilityNone
)

var (
	// ErrWalletNotUnique returned if another account has `wallet` field set to true.
	ErrWalletNotUnique = errors.New("another account is set to be default wallet. disable it before using new")
	// ErrChatNotUnique returned if another account has `chat` field set to true.
	ErrChatNotUnique = errors.New("another account is set to be default chat. disable it before using new")
	// ErrInvalidConfig returned if config isn't allowed
	ErrInvalidConfig = errors.New("configuration value not allowed")
)

type Account struct {
	Address   types.Address  `json:"address"`
	Wallet    bool           `json:"wallet"`
	Chat      bool           `json:"chat"`
	Type      string         `json:"type,omitempty"`
	Storage   string         `json:"storage,omitempty"`
	Path      string         `json:"path,omitempty"`
	PublicKey types.HexBytes `json:"public-key,omitempty"`
	Name      string         `json:"name"`
	Color     string         `json:"color"`
}

const (
	accountTypeGenerated = "generated"
	accountTypeKey       = "key"
	accountTypeSeed      = "seed"
	accountTypeWatch     = "watch"
)

// IsOwnAccount returns true if this is an account we have the private key for
// NOTE: Wallet flag can't be used as it actually indicates that it's the default
// Wallet
func (a *Account) IsOwnAccount() bool {
	return a.Wallet || a.Type == accountTypeSeed || a.Type == accountTypeGenerated || a.Type == accountTypeKey
}

type Settings struct {
	// required
	Address                   types.Address    `json:"address"`
	ChaosMode                 bool             `json:"chaos-mode?,omitempty"`
	Currency                  string           `json:"currency,omitempty"`
	CurrentNetwork            string           `json:"networks/current-network"`
	CustomBootnodes           *json.RawMessage `json:"custom-bootnodes,omitempty"`
	CustomBootnodesEnabled    *json.RawMessage `json:"custom-bootnodes-enabled?,omitempty"`
	DappsAddress              types.Address    `json:"dapps-address"`
	EIP1581Address            types.Address    `json:"eip1581-address"`
	Fleet                     *string          `json:"fleet,omitempty"`
	HideHomeTooltip           bool             `json:"hide-home-tooltip?,omitempty"`
	InstallationID            string           `json:"installation-id"`
	KeyUID                    string           `json:"key-uid"`
	KeycardInstanceUID        string           `json:"keycard-instance-uid,omitempty"`
	KeycardPAiredOn           int64            `json:"keycard-paired-on,omitempty"`
	KeycardPairing            string           `json:"keycard-pairing,omitempty"`
	LastUpdated               *int64           `json:"last-updated,omitempty"`
	LatestDerivedPath         uint             `json:"latest-derived-path"`
	LinkPreviewRequestEnabled bool             `json:"link-preview-request-enabled,omitempty"`
	LinkPreviewsEnabledSites  *json.RawMessage `json:"link-previews-enabled-sites,omitempty"`
	LogLevel                  *string          `json:"log-level,omitempty"`
	Mnemonic                  *string          `json:"mnemonic,omitempty"`
	Name                      string           `json:"name,omitempty"`
	Networks                  *json.RawMessage `json:"networks/networks"`
	// NotificationsEnabled indicates whether local notifications should be enabled (android only)
	NotificationsEnabled bool             `json:"notifications-enabled?,omitempty"`
	PhotoPath            string           `json:"photo-path"`
	PinnedMailserver     *json.RawMessage `json:"pinned-mailservers,omitempty"`
	PreferredName        *string          `json:"preferred-name,omitempty"`
	PreviewPrivacy       bool             `json:"preview-privacy?"`
	PublicKey            string           `json:"public-key"`
	// PushNotificationsServerEnabled indicates whether we should be running a push notification server
	PushNotificationsServerEnabled bool `json:"push-notifications-server-enabled?,omitempty"`
	// PushNotificationsFromContactsOnly indicates whether we should only receive push notifications from contacts
	PushNotificationsFromContactsOnly bool `json:"push-notifications-from-contacts-only?,omitempty"`
	// PushNotificationsBlockMentions indicates whether we should receive notifications for mentions
	PushNotificationsBlockMentions bool `json:"push-notifications-block-mentions?,omitempty"`
	RememberSyncingChoice          bool `json:"remember-syncing-choice?,omitempty"`
	// RemotePushNotificationsEnabled indicates whether we should be using remote notifications (ios only for now)
	RemotePushNotificationsEnabled bool             `json:"remote-push-notifications-enabled?,omitempty"`
	SigningPhrase                  string           `json:"signing-phrase"`
	StickerPacksInstalled          *json.RawMessage `json:"stickers/packs-installed,omitempty"`
	StickerPacksPending            *json.RawMessage `json:"stickers/packs-pending,omitempty"`
	StickersRecentStickers         *json.RawMessage `json:"stickers/recent-stickers,omitempty"`
	SyncingOnMobileNetwork         bool             `json:"syncing-on-mobile-network?,omitempty"`
	// SendPushNotifications indicates whether we should send push notifications for other clients
	SendPushNotifications bool `json:"send-push-notifications?,omitempty"`
	Appearance            uint `json:"appearance"`
	// ProfilePicturesVisibility indicates who we want to see profile pictures of (contacts, everyone or none)
	ProfilePicturesVisibility      uint             `json:"profile-pictures-visibility"`
	UseMailservers                 bool             `json:"use-mailservers?"`
	Usernames                      *json.RawMessage `json:"usernames,omitempty"`
	WalletRootAddress              types.Address    `json:"wallet-root-address,omitempty"`
	WalletSetUpPassed              bool             `json:"wallet-set-up-passed?,omitempty"`
	WalletVisibleTokens            *json.RawMessage `json:"wallet/visible-tokens,omitempty"`
	WakuBloomFilterMode            bool             `json:"waku-bloom-filter-mode,omitempty"`
	WebViewAllowPermissionRequests bool             `json:"webview-allow-permission-requests?,omitempty"`
}

func NewDB(db *sql.DB) *Database {
	return &Database{db: db}
}

// Database sql wrapper for operations with browser objects.
type Database struct {
	db *sql.DB
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
}

// TODO remove photoPath from settings
func (db *Database) CreateSettings(s Settings, nodecfg params.NodeConfig) error {
	_, err := db.db.Exec(`
INSERT INTO settings (
  address,
  currency,
  current_network,
  dapps_address,
  eip1581_address,
  installation_id,
  key_uid,
  keycard_instance_uid,
  keycard_paired_on,
  keycard_pairing,
  latest_derived_path,
  mnemonic,
  name,
  networks,
  node_config,
  photo_path,
  preview_privacy,
  public_key,
  signing_phrase,
  wallet_root_address,
  synthetic_id
) VALUES (
?,?,?,?,?,?,?,?,?,?,
?,?,?,?,?,?,?,?,?,?,
'id')`,
		s.Address,
		s.Currency,
		s.CurrentNetwork,
		s.DappsAddress,
		s.EIP1581Address,
		s.InstallationID,
		s.KeyUID,
		s.KeycardInstanceUID,
		s.KeycardPAiredOn,
		s.KeycardPairing,
		s.LatestDerivedPath,
		s.Mnemonic,
		s.Name,
		s.Networks,
		&sqlite.JSONBlob{nodecfg},
		s.PhotoPath,
		s.PreviewPrivacy,
		s.PublicKey,
		s.SigningPhrase,
		s.WalletRootAddress,
	)

	return err
}

func (db *Database) SaveSetting(setting string, value interface{}) error {
	var (
		update *sql.Stmt
		err    error
	)

	switch setting {
	case "chaos-mode?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET chaos_mode = ? WHERE synthetic_id = 'id'")
	case "currency":
		update, err = db.db.Prepare("UPDATE settings SET currency = ? WHERE synthetic_id = 'id'")
	case "custom-bootnodes":
		value = &sqlite.JSONBlob{value}
		update, err = db.db.Prepare("UPDATE settings SET custom_bootnodes = ? WHERE synthetic_id = 'id'")
	case "custom-bootnodes-enabled?":
		value = &sqlite.JSONBlob{value}
		update, err = db.db.Prepare("UPDATE settings SET custom_bootnodes_enabled = ? WHERE synthetic_id = 'id'")
	case "dapps-address":
		str, ok := value.(string)
		if ok {
			value = types.HexToAddress(str)
		} else {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET dapps_address = ? WHERE synthetic_id = 'id'")
	case "eip1581-address":
		str, ok := value.(string)
		if ok {
			value = types.HexToAddress(str)
		} else {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET eip1581_address = ? WHERE synthetic_id = 'id'")
	case "fleet":
		update, err = db.db.Prepare("UPDATE settings SET fleet = ? WHERE synthetic_id = 'id'")
	case "hide-home-tooltip?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET hide_home_tooltip = ? WHERE synthetic_id = 'id'")
	case "keycard-instance_uid":
		update, err = db.db.Prepare("UPDATE settings SET keycard_instance_uid = ? WHERE synthetic_id = 'id'")
	case "keycard-paired_on":
		update, err = db.db.Prepare("UPDATE settings SET keycard_paired_on = ? WHERE synthetic_id = 'id'")
	case "keycard-pairing":
		update, err = db.db.Prepare("UPDATE settings SET keycard_pairing = ? WHERE synthetic_id = 'id'")
	case "last-updated":
		update, err = db.db.Prepare("UPDATE settings SET last_updated = ? WHERE synthetic_id = 'id'")
	case "latest-derived-path":
		update, err = db.db.Prepare("UPDATE settings SET latest_derived_path = ? WHERE synthetic_id = 'id'")
	case "link-preview-request-enabled":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET link_preview_request_enabled = ? WHERE synthetic_id = 'id'")
	case "link-previews-enabled-sites":
		value = &sqlite.JSONBlob{value}
		update, err = db.db.Prepare("UPDATE settings SET link_previews_enabled_sites = ? WHERE synthetic_id = 'id'")
	case "log-level":
		update, err = db.db.Prepare("UPDATE settings SET log_level = ? WHERE synthetic_id = 'id'")
	case "mnemonic":
		update, err = db.db.Prepare("UPDATE settings SET mnemonic = ? WHERE synthetic_id = 'id'")
	case "name":
		update, err = db.db.Prepare("UPDATE settings SET name = ? WHERE synthetic_id = 'id'")
	case "networks/current-network":
		update, err = db.db.Prepare("UPDATE settings SET current_network = ? WHERE synthetic_id = 'id'")
	case "networks/networks":
		value = &sqlite.JSONBlob{value}
		update, err = db.db.Prepare("UPDATE settings SET networks = ? WHERE synthetic_id = 'id'")
	case "node-config":
		value = &sqlite.JSONBlob{value}
		update, err = db.db.Prepare("UPDATE settings SET node_config = ? WHERE synthetic_id = 'id'")
	case "notifications-enabled?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET notifications_enabled = ? WHERE synthetic_id = 'id'")
	case "photo-path":
		update, err = db.db.Prepare("UPDATE settings SET photo_path = ? WHERE synthetic_id = 'id'")
	case "pinned-mailservers":
		value = &sqlite.JSONBlob{value}
		update, err = db.db.Prepare("UPDATE settings SET pinned_mailservers = ? WHERE synthetic_id = 'id'")
	case "preferred-name":
		update, err = db.db.Prepare("UPDATE settings SET preferred_name = ? WHERE synthetic_id = 'id'")
	case "preview-privacy?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET preview_privacy = ? WHERE synthetic_id = 'id'")
	case "public-key":
		update, err = db.db.Prepare("UPDATE settings SET public_key = ? WHERE synthetic_id = 'id'")
	case "remember-syncing-choice?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET remember_syncing_choice = ? WHERE synthetic_id = 'id'")
	case "remote-push-notifications-enabled?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET remote_push_notifications_enabled = ? WHERE synthetic_id = 'id'")
	case "push-notifications-server-enabled?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET push_notifications_server_enabled = ? WHERE synthetic_id = 'id'")
	case "push-notifications-from-contacts-only?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET push_notifications_from_contacts_only = ? WHERE synthetic_id = 'id'")
	case "push-notifications-block-mentions?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET push_notifications_block_mentions = ? WHERE synthetic_id = 'id'")
	case "send-push-notifications?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET send_push_notifications = ? WHERE synthetic_id = 'id'")
	case "stickers/packs-installed":
		value = &sqlite.JSONBlob{value}
		update, err = db.db.Prepare("UPDATE settings SET stickers_packs_installed = ? WHERE synthetic_id = 'id'")
	case "stickers/packs-pending":
		value = &sqlite.JSONBlob{value}
		update, err = db.db.Prepare("UPDATE settings SET stickers_packs_pending = ? WHERE synthetic_id = 'id'")
	case "stickers/recent-stickers":
		value = &sqlite.JSONBlob{value}
		update, err = db.db.Prepare("UPDATE settings SET stickers_recent_stickers = ? WHERE synthetic_id = 'id'")
	case "syncing-on-mobile-network?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET syncing_on_mobile_network = ? WHERE synthetic_id = 'id'")
	case "use-mailservers?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET use_mailservers = ? WHERE synthetic_id = 'id'")
	case "usernames":
		value = &sqlite.JSONBlob{value}
		update, err = db.db.Prepare("UPDATE settings SET usernames = ? WHERE synthetic_id = 'id'")
	case "wallet-set-up-passed?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET wallet_set_up_passed = ? WHERE synthetic_id = 'id'")
	case "wallet/visible-tokens":
		value = &sqlite.JSONBlob{value}
		update, err = db.db.Prepare("UPDATE settings SET wallet_visible_tokens = ? WHERE synthetic_id = 'id'")
	case "appearance":
		update, err = db.db.Prepare("UPDATE settings SET appearance = ? WHERE synthetic_id = 'id'")
	case "profile-pictures-visibility":
		update, err = db.db.Prepare("UPDATE settings SET profile_pictures_visibility = ? WHERE synthetic_id = 'id'")
	case "waku-bloom-filter-mode":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET waku_bloom_filter_mode = ? WHERE synthetic_id = 'id'")
	case "webview-allow-permission-requests?":
		_, ok := value.(bool)
		if !ok {
			return ErrInvalidConfig
		}
		update, err = db.db.Prepare("UPDATE settings SET webview_allow_permission_requests = ? WHERE synthetic_id = 'id'")
	default:
		return ErrInvalidConfig
	}
	if err != nil {
		return err
	}
	_, err = update.Exec(value)
	return err
}

func (db *Database) GetNodeConfig(nodecfg interface{}) error {
	return db.db.QueryRow("SELECT node_config FROM settings WHERE synthetic_id = 'id'").Scan(&sqlite.JSONBlob{nodecfg})
}

func (db *Database) GetSettings() (Settings, error) {
	var s Settings
	err := db.db.QueryRow("SELECT address, chaos_mode, currency, current_network, custom_bootnodes, custom_bootnodes_enabled, dapps_address, eip1581_address, fleet, hide_home_tooltip, installation_id, key_uid, keycard_instance_uid, keycard_paired_on, keycard_pairing, last_updated, latest_derived_path, link_preview_request_enabled, link_previews_enabled_sites, log_level, mnemonic, name, networks, notifications_enabled, push_notifications_server_enabled, push_notifications_from_contacts_only, remote_push_notifications_enabled, send_push_notifications, push_notifications_block_mentions, photo_path, pinned_mailservers, preferred_name, preview_privacy, public_key, remember_syncing_choice, signing_phrase, stickers_packs_installed, stickers_packs_pending, stickers_recent_stickers, syncing_on_mobile_network, use_mailservers, usernames, appearance, profile_pictures_visibility, wallet_root_address, wallet_set_up_passed, wallet_visible_tokens, waku_bloom_filter_mode, webview_allow_permission_requests FROM settings WHERE synthetic_id = 'id'").Scan(
		&s.Address,
		&s.ChaosMode,
		&s.Currency,
		&s.CurrentNetwork,
		&s.CustomBootnodes,
		&s.CustomBootnodesEnabled,
		&s.DappsAddress,
		&s.EIP1581Address,
		&s.Fleet,
		&s.HideHomeTooltip,
		&s.InstallationID,
		&s.KeyUID,
		&s.KeycardInstanceUID,
		&s.KeycardPAiredOn,
		&s.KeycardPairing,
		&s.LastUpdated,
		&s.LatestDerivedPath,
		&s.LinkPreviewRequestEnabled,
		&s.LinkPreviewsEnabledSites,
		&s.LogLevel,
		&s.Mnemonic,
		&s.Name,
		&s.Networks,
		&s.NotificationsEnabled,
		&s.PushNotificationsServerEnabled,
		&s.PushNotificationsFromContactsOnly,
		&s.RemotePushNotificationsEnabled,
		&s.SendPushNotifications,
		&s.PushNotificationsBlockMentions,
		&s.PhotoPath,
		&s.PinnedMailserver,
		&s.PreferredName,
		&s.PreviewPrivacy,
		&s.PublicKey,
		&s.RememberSyncingChoice,
		&s.SigningPhrase,
		&s.StickerPacksInstalled,
		&s.StickerPacksPending,
		&s.StickersRecentStickers,
		&s.SyncingOnMobileNetwork,
		&s.UseMailservers,
		&s.Usernames,
		&s.Appearance,
		&s.ProfilePicturesVisibility,
		&s.WalletRootAddress,
		&s.WalletSetUpPassed,
		&s.WalletVisibleTokens,
		&s.WakuBloomFilterMode,
		&s.WebViewAllowPermissionRequests)
	return s, err
}

func (db *Database) GetAccounts() ([]Account, error) {
	rows, err := db.db.Query("SELECT address, wallet, chat, type, storage, pubkey, path, name, color FROM accounts ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := []Account{}
	pubkey := []byte{}
	for rows.Next() {
		acc := Account{}
		err := rows.Scan(
			&acc.Address, &acc.Wallet, &acc.Chat, &acc.Type, &acc.Storage,
			&pubkey, &acc.Path, &acc.Name, &acc.Color)
		if err != nil {
			return nil, err
		}
		if lth := len(pubkey); lth > 0 {
			acc.PublicKey = make(types.HexBytes, lth)
			copy(acc.PublicKey, pubkey)
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

func (db *Database) GetAccountByAddress(address types.Address) (rst *Account, err error) {
	row := db.db.QueryRow("SELECT address, wallet, chat, type, storage, pubkey, path, name, color FROM accounts  WHERE address = ? COLLATE NOCASE", address)

	acc := &Account{}
	pubkey := []byte{}
	err = row.Scan(
		&acc.Address, &acc.Wallet, &acc.Chat, &acc.Type, &acc.Storage,
		&pubkey, &acc.Path, &acc.Name, &acc.Color)

	if err != nil {
		return nil, err
	}
	acc.PublicKey = pubkey
	return acc, nil
}

func (db *Database) SaveAccounts(accounts []Account) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
		update *sql.Stmt
	)
	tx, err = db.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()
	// NOTE(dshulyak) replace all record values using address (primary key)
	// can't use `insert or replace` because of the additional constraints (wallet and chat)
	insert, err = tx.Prepare("INSERT OR IGNORE INTO accounts (address, created_at, updated_at) VALUES (?, datetime('now'), datetime('now'))")
	if err != nil {
		return err
	}
	update, err = tx.Prepare("UPDATE accounts SET wallet = ?, chat = ?, type = ?, storage = ?, pubkey = ?, path = ?, name = ?, color = ?, updated_at = datetime('now') WHERE address = ?")
	if err != nil {
		return err
	}
	for i := range accounts {
		acc := &accounts[i]
		_, err = insert.Exec(acc.Address)
		if err != nil {
			return
		}
		_, err = update.Exec(acc.Wallet, acc.Chat, acc.Type, acc.Storage, acc.PublicKey, acc.Path, acc.Name, acc.Color, acc.Address)
		if err != nil {
			switch err.Error() {
			case uniqueChatConstraint:
				err = ErrChatNotUnique
			case uniqueWalletConstraint:
				err = ErrWalletNotUnique
			}
			return
		}
	}
	return
}

func (db *Database) DeleteAccount(address types.Address) error {
	_, err := db.db.Exec("DELETE FROM accounts WHERE address = ?", address)
	return err
}

func (db *Database) GetNotificationsEnabled() (bool, error) {
	var result bool
	err := db.db.QueryRow("SELECT notifications_enabled FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return result, nil
	}
	return result, err
}

func (db *Database) GetWalletAddress() (rst types.Address, err error) {
	err = db.db.QueryRow("SELECT address FROM accounts WHERE wallet = 1").Scan(&rst)
	return
}

func (db *Database) GetWalletAddresses() (rst []types.Address, err error) {
	rows, err := db.db.Query("SELECT address FROM accounts WHERE chat = 0 ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		addr := types.Address{}
		err = rows.Scan(&addr)
		if err != nil {
			return nil, err
		}
		rst = append(rst, addr)
	}
	return rst, nil
}

func (db *Database) GetChatAddress() (rst types.Address, err error) {
	err = db.db.QueryRow("SELECT address FROM accounts WHERE chat = 1").Scan(&rst)
	return
}

func (db *Database) GetAddresses() (rst []types.Address, err error) {
	rows, err := db.db.Query("SELECT address FROM accounts ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		addr := types.Address{}
		err = rows.Scan(&addr)
		if err != nil {
			return nil, err
		}
		rst = append(rst, addr)
	}
	return rst, nil
}

// AddressExists returns true if given address is stored in database.
func (db *Database) AddressExists(address types.Address) (exists bool, err error) {
	err = db.db.QueryRow("SELECT EXISTS (SELECT 1 FROM accounts WHERE address = ?)", address).Scan(&exists)
	return exists, err
}
