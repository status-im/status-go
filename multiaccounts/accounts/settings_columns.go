package accounts

type SettingField struct {
	ReactFieldName string
	DBColumnName   string
}

var (
	Currency                  = SettingField{"currency", "currency"}
	GifRecents                = SettingField{"gifs/recent-gifs", "gif_recents"}
	GifFavourites             = SettingField{"gifs/favorite-gifs", "gif_favorites"}
	MessagesFromContactsOnly  = SettingField{"messages-from-contacts-only", "messages_from_contacts_only"}
	PreferredName             = SettingField{"preferred-name", "preferred_name"}
	PreviewPrivacy            = SettingField{"preview-privacy?", "preview_privacy"}
	ProfilePicturesShowTo     = SettingField{"profile-pictures-show-to", "profile_pictures_show_to"}
	ProfilePicturesVisibility = SettingField{"profile-pictures-visibility", "profile_pictures_visibility"}
	SendStatusUpdates         = SettingField{"send-status-updates?", "send_status_updates"}
	StickersPacksInstalled    = SettingField{"stickers/packs-installed", "stickers_packs_installed"}
	StickersPacksPending      = SettingField{"stickers/packs-pending", "stickers_packs_pending"}
	StickersRecentStickers    = SettingField{"stickers/recent-stickers", "stickers_recent_stickers"}
	TelemetryServerURL        = SettingField{"telemetry-server-url", "telemetry_server_url"}

	SettingFields = []SettingField{
		Currency,
		GifRecents,
		GifFavourites,
		MessagesFromContactsOnly,
		PreferredName,
		PreviewPrivacy,
		ProfilePicturesShowTo,
		ProfilePicturesVisibility,
		SendStatusUpdates,
		StickersPacksInstalled,
		StickersPacksPending,
		StickersRecentStickers,
		TelemetryServerURL,
	}
)
