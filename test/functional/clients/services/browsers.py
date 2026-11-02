from clients.rpc import RpcClient
from clients.services.service import Service


class BrowsersService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "browsers")

    def get_bookmarks(self):
        return self.rpc_request("getBookmarks")

    def store_bookmark(self, bookmark: dict):
        return self.rpc_request("storeBookmark", [bookmark])

    def update_bookmark(self, original_url: str, bookmark: dict):
        return self.rpc_request("updateBookmark", [original_url, bookmark])

    def remove_bookmark(self, url: str):
        return self.rpc_request("removeBookmark", [url])

    def delete_bookmark(self, url: str):
        return self.rpc_request("deleteBookmark", [url])
