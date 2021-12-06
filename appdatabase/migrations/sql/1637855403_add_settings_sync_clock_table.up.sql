CREATE TABLE settings_sync_clock (
    currency TIMESTAMP NOT NULL,
    gif_recents TIMESTAMP NOT NULL,
    gif_favorites TIMESTAMP NOT NULL,
    messages_from_contacts_only TIMESTAMP NOT NULL,
    preferred_name TIMESTAMP NOT NULL,
    preview_privacy TIMESTAMP NOT NULL,
    profile_pictures_show_to TIMESTAMP NOT NULL,
    profile_pictures_visibility TIMESTAMP NOT NULL,
    send_status_updates TIMESTAMP NOT NULL,
    stickers_packs_installed TIMESTAMP NOT NULL,
    stickers_packs_pending TIMESTAMP NOT NULL,
    stickers_recent_stickers TIMESTAMP NOT NULL,
    telemetry_server_url TIMESTAMP NOT NULL,
    synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;
INSERT INTO settings_sync_clock (currency, gif_recents, gif_favorites, messages_from_contacts_only, preferred_name, preview_privacy, profile_pictures_show_to, profile_pictures_visibility, send_status_updates, stickers_packs_installed, stickers_packs_pending, stickers_recent_stickers, telemetry_server_url) VALUES (0,0,0,0,0,0,0,0,0,0,0,0,0);