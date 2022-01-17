CREATE TABLE settings_sync_clock (
    currency INTEGER NOT NULL DEFAULT 0,
    gif_recents INTEGER NOT NULL DEFAULT 0,
    gif_favorites INTEGER NOT NULL DEFAULT 0,
    messages_from_contacts_only INTEGER NOT NULL DEFAULT 0,
    preferred_name INTEGER NOT NULL DEFAULT 0,
    preview_privacy INTEGER NOT NULL DEFAULT 0,
    profile_pictures_show_to INTEGER NOT NULL DEFAULT 0,
    profile_pictures_visibility INTEGER NOT NULL DEFAULT 0,
    send_status_updates INTEGER NOT NULL DEFAULT 0,
    stickers_packs_installed INTEGER NOT NULL DEFAULT 0,
    stickers_packs_pending INTEGER NOT NULL DEFAULT 0,
    stickers_recent_stickers INTEGER NOT NULL DEFAULT 0,
    telemetry_server_url INTEGER NOT NULL DEFAULT 0,
    synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;
INSERT INTO settings_sync_clock (currency, gif_recents, gif_favorites, messages_from_contacts_only, preferred_name, preview_privacy, profile_pictures_show_to, profile_pictures_visibility, send_status_updates, stickers_packs_installed, stickers_packs_pending, stickers_recent_stickers, telemetry_server_url) VALUES (0,0,0,0,0,0,0,0,0,0,0,0,0);