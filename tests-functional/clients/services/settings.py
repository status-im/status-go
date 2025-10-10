from clients.rpc import RpcClient
from clients.services.service import Service


class SettingsService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "settings")

    def get_settings(self):
        response = self.rpc_request("getSettings")
        return response

    def save_setting(self, key, value):
        params = [key, value]
        response = self.rpc_request("saveSetting", params)
        return response

    def get_node_config(self):
        return self.rpc_request("nodeConfig")

    def news_feed_enabled(self):
        return self.rpc_request("newsFeedEnabled")

    def news_notifications_enabled(self):
        return self.rpc_request("newsNotificationsEnabled")

    def news_rss_enabled(self):
        return self.rpc_request("newsRSSEnabled")

    def backup_path(self):
        return self.rpc_request("backupPath")

    def messages_backup_enabled(self):
        return self.rpc_request("messagesBackupEnabled")

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
