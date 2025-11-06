from clients.rpc import RpcClient
from clients.services.service import Service


class NewsFeedService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "newsfeed")

    def enabled(self) -> bool:
        """Check if newsfeed is enabled."""
        return self.rpc_request("enabled")

    def set_enabled(self, value: bool) -> None:
        """Enable or disable newsfeed."""
        params = [value]
        self.rpc_request("setEnabled", params)

    def notifications_enabled(self) -> bool:
        """Check if notifications are enabled."""
        return self.rpc_request("notificationsEnabled")

    def set_notifications_enabled(self, value: bool) -> None:
        """Enable or disable notifications."""
        params = [value]
        self.rpc_request("setNotificationsEnabled", params)

    def rss_enabled(self) -> bool:
        """Check if RSS is enabled."""
        return self.rpc_request("rSSEnabled")

    def set_rss_enabled(self, value: bool) -> None:
        """Enable or disable RSS."""
        params = [value]
        self.rpc_request("setRSSEnabled", params)
