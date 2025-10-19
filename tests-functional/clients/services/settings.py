from clients.rpc import RpcClient
from clients.services.service import Service


class SettingsService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "settings")

    # Base settings
    def get_settings(self):
        return self.rpc_request("getSettings")

    def save_setting(self, key, value):
        params = [key, value]
        return self.rpc_request("saveSetting", params)

    def get_node_config(self):
        return self.rpc_request("nodeConfig")

    # News Settings
    def news_feed_enabled(self):
        return self.rpc_request("newsFeedEnabled")

    def news_notifications_enabled(self):
        return self.rpc_request("newsNotificationsEnabled")

    def news_rss_enabled(self):
        return self.rpc_request("newsRSSEnabled")

    # Backup Settings
    def backup_path(self):
        return self.rpc_request("backupPath")

    def messages_backup_enabled(self):
        return self.rpc_request("messagesBackupEnabled")

    # Notifications Settings
    def notifications_get_allow_notifications(self):
        return self.rpc_request("notificationsGetAllowNotifications")

    def notifications_set_allow_notifications(self, value: bool):
        params = [value]
        return self.rpc_request("notificationsSetAllowNotifications", params)

    def notifications_get_one_to_one_chats(self):
        return self.rpc_request("notificationsGetOneToOneChats")

    def notifications_set_one_to_one_chats(self, value: str):
        params = [value]
        return self.rpc_request("notificationsSetOneToOneChats", params)

    def notifications_get_group_chats(self):
        return self.rpc_request("notificationsGetGroupChats")

    def notifications_set_group_chats(self, value: str):
        params = [value]
        return self.rpc_request("notificationsSetGroupChats", params)

    def notifications_get_personal_mentions(self):
        return self.rpc_request("notificationsGetPersonalMentions")

    def notifications_set_personal_mentions(self, value: str):
        params = [value]
        return self.rpc_request("notificationsSetPersonalMentions", params)

    def notifications_get_global_mentions(self):
        return self.rpc_request("notificationsGetGlobalMentions")

    def notifications_set_global_mentions(self, value: str):
        params = [value]
        return self.rpc_request("notificationsSetGlobalMentions", params)

    def notifications_get_all_messages(self):
        return self.rpc_request("notificationsGetAllMessages")

    def notifications_set_all_messages(self, value: str):
        params = [value]
        return self.rpc_request("notificationsSetAllMessages", params)

    def notifications_get_contact_requests(self):
        return self.rpc_request("notificationsGetContactRequests")

    def notifications_set_contact_requests(self, value: str):
        params = [value]
        return self.rpc_request("notificationsSetContactRequests", params)

    def notifications_get_identity_verification_requests(self):
        return self.rpc_request("notificationsGetIdentityVerificationRequests")

    def notifications_set_identity_verification_requests(self, value: str):
        params = [value]
        return self.rpc_request("notificationsSetIdentityVerificationRequests", params)

    def notifications_get_sound_enabled(self):
        return self.rpc_request("notificationsGetSoundEnabled")

    def notifications_set_sound_enabled(self, value: bool):
        params = [value]
        return self.rpc_request("notificationsSetSoundEnabled", params)

    def notifications_get_volume(self):
        return self.rpc_request("notificationsGetVolume")

    def notifications_set_volume(self, value: int):
        params = [value]
        return self.rpc_request("notificationsSetVolume", params)

    def notifications_get_message_preview(self):
        return self.rpc_request("notificationsGetMessagePreview")

    def notifications_set_message_preview(self, value: int):
        params = [value]
        return self.rpc_request("notificationsSetMessagePreview", params)

    def notifications_get_ex_mute_all_messages(self, id: str):
        params = [id]
        return self.rpc_request("notificationsGetExMuteAllMessages", params)

    def notifications_get_ex_personal_mentions(self, id: str):
        params = [id]
        return self.rpc_request("notificationsGetExPersonalMentions", params)

    def notifications_get_ex_global_mentions(self, id: str):
        params = [id]
        return self.rpc_request("notificationsGetExGlobalMentions", params)

    def notifications_get_ex_other_messages(self, id: str):
        params = [id]
        return self.rpc_request("notificationsGetExOtherMessages", params)

    def notifications_set_exemptions(self, id: str, mute_all_messages: bool, personal_mentions: str, global_mentions: str, other_messages: str):
        params = [id, mute_all_messages, personal_mentions, global_mentions, other_messages]
        return self.rpc_request("notificationsSetExemptions", params)

    def delete_exemptions(self, id: str):
        params = [id]
        return self.rpc_request("deleteExemptions", params)

    def set_bio(self, bio: str):
        params = [bio]
        return self.rpc_request("setBio", params)

    def mnemonic_was_shown(self):
        return self.rpc_request("mnemonicWasShown")

    def last_tokens_update(self):
        return self.rpc_request("lastTokensUpdate")

    def thirdparty_services_enabled(self):
        return self.rpc_request("thirdpartyServicesEnabled")

    def notifications_get_default_exemptions(self):
        return self.rpc_request("notificationsGetDefaultExemptions")
