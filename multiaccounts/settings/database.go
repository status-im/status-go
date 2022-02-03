package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/errors"
	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/sqlite"
)

var (
	// dbInstances holds a map of singleton instances of Database
	dbInstances map[string]*Database

	// mutex guards the instantiation of the dbInstances values, to prevent any concurrent instantiations
	mutex sync.Mutex
)

// Database sql wrapper for operations with browser objects.
type Database struct {
	db        *sql.DB
	SyncQueue chan SyncSettingField
}

// MakeNewDB ensures that a singleton instance of Database is returned per sqlite db file
func MakeNewDB(db *sql.DB) (*Database, error) {
	filename, err := appdatabase.GetDBFilename(db)
	if err != nil {
		return nil, err
	}

	d := &Database{
		db:        db,
		SyncQueue: make(chan SyncSettingField, 100),
	}

	// An empty filename means that the sqlite database is held in memory
	// In this case we don't want to restrict the instantiation
	if filename == "" {
		return d, nil
	}

	// Lock to protect the map from concurrent access
	mutex.Lock()
	defer mutex.Unlock()

	// init dbInstances if it hasn't been already
	if dbInstances == nil {
		dbInstances = map[string]*Database{}
	}

	// If we haven't seen this database file before make an instance
	if _, ok := dbInstances[filename]; !ok {
		dbInstances[filename] = d
	}

	return dbInstances[filename], nil
}

// TODO remove photoPath from settings
func (db *Database) CreateSettings(s Settings, n params.NodeConfig) error {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	_, err = tx.Exec(`
INSERT INTO settings (
  address,
  currency,
  current_network,
  dapps_address,
  display_name,
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
  photo_path,
  preview_privacy,
  public_key,
  signing_phrase,
  wallet_root_address,
  synthetic_id
) VALUES (
?,?,?,?,?,?,?,?,?,?,?,
?,?,?,?,?,?,?,?,?,'id')`,
		s.Address,
		s.Currency,
		s.CurrentNetwork,
		s.DappsAddress,
		s.DisplayName,
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
		s.PhotoPath,
		s.PreviewPrivacy,
		s.PublicKey,
		s.SigningPhrase,
		s.WalletRootAddress,
	)
	if err != nil {
		return err
	}

	return nodecfg.SaveConfigWithTx(tx, &n)
}

func (db *Database) getSettingFieldFromReactName(reactName string) (SettingField, error) {
	for _, s := range SettingFieldRegister {
		if s.GetReactName() == reactName {
			return s, nil
		}
	}
	return SettingField{}, errors.ErrInvalidConfig
}

func (db *Database) saveSetting(setting SettingField, value interface{}) error {
	query := "UPDATE settings SET %s = ? WHERE synthetic_id = 'id'"
	query = fmt.Sprintf(query, setting.GetDBName())
	update, err := db.db.Prepare(query)
	if err != nil {
		return err
	}

	if setting.ValueHandler() != nil {
		value, err = setting.ValueHandler()(value)
		if err != nil {
			return err
		}
	}

	// TODO(samyoul) this is ugly as hell need a more elegant solution
	if NodeConfig.GetReactName() == setting.GetReactName() {
		if err = nodecfg.SaveNodeConfig(db.db, value.(*params.NodeConfig)); err != nil {
			return err
		}
		value = nil
	}

	_, err = update.Exec(value)
	return err
}

// SaveSetting stores data from any non-sync source
// If the field requires syncing the field data is pushed on to the SyncQueue
func (db *Database) SaveSetting(setting string, value interface{}) error {
	sf, err := db.getSettingFieldFromReactName(setting)
	if err != nil {
		return err
	}

	err = db.saveSetting(sf, value)
	if err != nil {
		return err
	}

	if sf.SyncProtobufFactory() != nil {
		db.SyncQueue <- SyncSettingField{
			Field: sf,
			Value: value,
		}
	}
	return nil
}

// SaveSyncSetting stores setting data from a sync protobuf source
func (db *Database) SaveSyncSetting(setting SettingField, value interface{}, clock uint64) error {
	ls, err := db.GetSettingLastSynced(setting)
	if err != nil {
		return err
	}
	if clock < ls {
		return errors.ErrNewClockOlderThanCurrent
	}

	err = db.setSettingLastSynced(setting, clock)
	if err != nil {
		return err
	}

	return db.saveSetting(setting, value)
}

func (db *Database) GetSettings() (Settings, error) {
	var s Settings
	err := db.db.QueryRow("SELECT address, anon_metrics_should_send, chaos_mode, currency, current_network, custom_bootnodes, custom_bootnodes_enabled, dapps_address, display_name, eip1581_address, fleet, hide_home_tooltip, installation_id, key_uid, keycard_instance_uid, keycard_paired_on, keycard_pairing, last_updated, latest_derived_path, link_preview_request_enabled, link_previews_enabled_sites, log_level, mnemonic, name, networks, notifications_enabled, push_notifications_server_enabled, push_notifications_from_contacts_only, remote_push_notifications_enabled, send_push_notifications, push_notifications_block_mentions, photo_path, pinned_mailservers, preferred_name, preview_privacy, public_key, remember_syncing_choice, signing_phrase, stickers_packs_installed, stickers_packs_pending, stickers_recent_stickers, syncing_on_mobile_network, default_sync_period, use_mailservers, messages_from_contacts_only, usernames, appearance, profile_pictures_show_to, profile_pictures_visibility, wallet_root_address, wallet_set_up_passed, wallet_visible_tokens, waku_bloom_filter_mode, webview_allow_permission_requests, current_user_status, send_status_updates, gif_recents, gif_favorites, opensea_enabled, last_backup, backup_enabled, telemetry_server_url, auto_message_enabled, gif_api_key FROM settings WHERE synthetic_id = 'id'").Scan(
		&s.Address,
		&s.AnonMetricsShouldSend,
		&s.ChaosMode,
		&s.Currency,
		&s.CurrentNetwork,
		&s.CustomBootnodes,
		&s.CustomBootnodesEnabled,
		&s.DappsAddress,
		&s.DisplayName,
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
		&s.DefaultSyncPeriod,
		&s.UseMailservers,
		&s.MessagesFromContactsOnly,
		&s.Usernames,
		&s.Appearance,
		&s.ProfilePicturesShowTo,
		&s.ProfilePicturesVisibility,
		&s.WalletRootAddress,
		&s.WalletSetUpPassed,
		&s.WalletVisibleTokens,
		&s.WakuBloomFilterMode,
		&s.WebViewAllowPermissionRequests,
		&sqlite.JSONBlob{Data: &s.CurrentUserStatus},
		&s.SendStatusUpdates,
		&sqlite.JSONBlob{Data: &s.GifRecents},
		&sqlite.JSONBlob{Data: &s.GifFavorites},
		&s.OpenseaEnabled,
		&s.LastBackup,
		&s.BackupEnabled,
		&s.TelemetryServerURL,
		&s.AutoMessageEnabled,
		&s.GifAPIKey,
	)

	return s, err
}

func (db *Database) GetNotificationsEnabled() (bool, error) {
	var result bool
	err := db.db.QueryRow("SELECT notifications_enabled FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return result, nil
	}
	return result, err
}

func (db *Database) GetProfilePicturesVisibility() (int, error) {
	var result int
	err := db.db.QueryRow("SELECT profile_pictures_visibility FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return result, nil
	}
	return result, err
}

func (db *Database) GetPublicKey() (rst string, err error) {
	err = db.db.QueryRow("SELECT public_key FROM settings WHERE synthetic_id = 'id'").Scan(&rst)
	if err == sql.ErrNoRows {
		return rst, nil
	}
	return
}

func (db *Database) GetFleet() (rst string, err error) {
	err = db.db.QueryRow("SELECT COALESCE(fleet, '') FROM settings WHERE synthetic_id = 'id'").Scan(&rst)
	if err == sql.ErrNoRows {
		return rst, nil
	}
	return
}

func (db *Database) GetDappsAddress() (rst types.Address, err error) {
	err = db.db.QueryRow("SELECT dapps_address FROM settings WHERE synthetic_id = 'id'").Scan(&rst)
	if err == sql.ErrNoRows {
		return rst, nil
	}
	return
}

func (db *Database) GetPinnedMailservers() (rst map[string]string, err error) {
	rst = make(map[string]string)
	var pinnedMailservers string
	err = db.db.QueryRow("SELECT COALESCE(pinned_mailservers, '') FROM settings WHERE synthetic_id = 'id'").Scan(&pinnedMailservers)
	if err == sql.ErrNoRows || pinnedMailservers == "" {
		return rst, nil
	}

	err = json.Unmarshal([]byte(pinnedMailservers), &rst)
	if err != nil {
		return nil, err
	}
	return
}

func (db *Database) CanUseMailservers() (bool, error) {
	var result bool
	err := db.db.QueryRow("SELECT use_mailservers FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return result, nil
	}
	return result, err
}

func (db *Database) CanSyncOnMobileNetwork() (bool, error) {
	var result bool
	err := db.db.QueryRow("SELECT syncing_on_mobile_network FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return result, nil
	}
	return result, err
}

func (db *Database) GetDefaultSyncPeriod() (uint32, error) {
	var result uint32
	err := db.db.QueryRow("SELECT default_sync_period FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return result, nil
	}
	return result, err
}

func (db *Database) GetMessagesFromContactsOnly() (bool, error) {
	var result bool
	err := db.db.QueryRow("SELECT messages_from_contacts_only FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return result, nil
	}
	return result, err
}

func (db *Database) GetLatestDerivedPath() (uint, error) {
	var result uint
	err := db.db.QueryRow("SELECT latest_derived_path FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	return result, err
}

func (db *Database) GetCurrentStatus(status interface{}) error {
	err := db.db.QueryRow("SELECT current_user_status FROM settings WHERE synthetic_id = 'id'").Scan(&sqlite.JSONBlob{Data: &status})
	if err == sql.ErrNoRows {
		return nil
	}
	return err
}

func (db *Database) ShouldBroadcastUserStatus() (bool, error) {
	var result bool
	err := db.db.QueryRow("SELECT send_status_updates FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	// If the `send_status_updates` value is nil the sql.ErrNoRows will be returned
	// because this feature is opt out, `true` should be returned in the case where no value is found
	if err == sql.ErrNoRows {
		return true, nil
	}
	return result, err
}

func (db *Database) BackupEnabled() (bool, error) {
	var result bool
	err := db.db.QueryRow("SELECT backup_enabled FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return true, nil
	}
	return result, err
}

func (db *Database) AutoMessageEnabled() (bool, error) {
	var result bool
	err := db.db.QueryRow("SELECT auto_message_enabled FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return true, nil
	}
	return result, err
}

func (db *Database) LastBackup() (uint64, error) {
	var result uint64
	err := db.db.QueryRow("SELECT last_backup FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return result, err
}

func (db *Database) SetLastBackup(time uint64) error {
	_, err := db.db.Exec("UPDATE settings SET last_backup = ?", time)
	return err
}

func (db *Database) SetBackupFetched(fetched bool) error {
	_, err := db.db.Exec("UPDATE settings SET backup_fetched = ?", fetched)
	return err
}

func (db *Database) BackupFetched() (bool, error) {
	var result bool
	err := db.db.QueryRow("SELECT backup_fetched FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return true, nil
	}
	return result, err
}

func (db *Database) ENSName() (string, error) {
	var result sql.NullString
	err := db.db.QueryRow("SELECT preferred_name FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if result.Valid {
		return result.String, nil
	}
	return "", err
}

func (db *Database) DisplayName() (string, error) {
	var result sql.NullString
	err := db.db.QueryRow("SELECT display_name FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if result.Valid {
		return result.String, nil
	}
	return "", err
}

func (db *Database) GifAPIKey() (string, error) {
	var result sql.NullString
	err := db.db.QueryRow("SELECT gif_api_key FROM settings WHERE synthetic_id = 'id'").Scan(&result)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if result.Valid {
		return result.String, nil
	}
	return "", err
}

func (db *Database) GifRecents() (recents json.RawMessage, err error) {
	err = db.db.QueryRow("SELECT gif_recents FROM settings WHERE synthetic_id = 'id'").Scan(&sqlite.JSONBlob{Data: &recents})
	if err == sql.ErrNoRows {
		return nil, err
	}
	return recents, nil
}

func (db *Database) GifFavorites() (favorites json.RawMessage, err error) {
	err = db.db.QueryRow("SELECT gif_favorites FROM settings WHERE synthetic_id = 'id'").Scan(&sqlite.JSONBlob{Data: &favorites})
	if err == sql.ErrNoRows {
		return nil, err
	}
	return favorites, nil
}

func (db *Database) GetSettingLastSynced(setting SettingField) (uint64, error) {
	var result uint64

	query := "SELECT %s FROM settings_sync_clock WHERE synthetic_id = 'id'"
	query = fmt.Sprintf(query, setting.GetDBName())

	err := db.db.QueryRow(query).Scan(&result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func (db *Database) setSettingLastSynced(setting SettingField, clock uint64) error {
	query := "UPDATE settings_sync_clock SET %s = ? WHERE synthetic_id = 'id' AND %s < ?"
	query = fmt.Sprintf(query, setting.GetDBName(), setting.GetDBName())

	_, err := db.db.Exec(query, clock, clock)
	return err
}

func (db *Database) GetPreferredUsername() (rst string, err error) {
	err = db.db.QueryRow("SELECT preferred_name FROM settings WHERE synthetic_id = 'id'").Scan(&rst)
	if err == sql.ErrNoRows {
		return rst, nil
	}
	return
}

func (db *Database) GetInstalledStickerPacks() (rst *json.RawMessage, err error) {
	err = db.db.QueryRow("SELECT stickers_packs_installed FROM settings WHERE synthetic_id = 'id'").Scan(&rst)
	return
}

func (db *Database) GetPendingStickerPacks() (rst *json.RawMessage, err error) {
	err = db.db.QueryRow("SELECT stickers_packs_pending FROM settings WHERE synthetic_id = 'id'").Scan(&rst)
	return
}

func (db *Database) GetRecentStickers() (rst *json.RawMessage, err error) {
	err = db.db.QueryRow("SELECT stickers_recent_stickers FROM settings WHERE synthetic_id = 'id'").Scan(&rst)
	return
}

func (db *Database) SetPinnedMailservers(mailservers map[string]string) error {
	jsonString, err := json.Marshal(mailservers)
	if err != nil {
		return err
	}

	_, err = db.db.Exec("UPDATE settings SET pinned_mailservers = ? WHERE synthetic_id = 'id'", jsonString)
	return err
}

func (db *Database) SetUseMailservers(value bool) error {
	_, err := db.db.Exec("UPDATE settings SET use_mailservers = ?", value)
	return err
}
