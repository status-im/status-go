from clients.rpc import RpcClient
from clients.services.service import Service

from enum import IntEnum


class URLUnfurlPermission(IntEnum):
    URLUnfurlingAllowed = 0
    URLUnfurlingAskUser = 1
    URLUnfurlingForbiddenBySettings = 2
    URLUnfurlingNotSupported = 3


class LinkPreviewService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "linkpreview")

    def get_text_urls_to_unfurl(self, text: str):
        params = [text]
        response = self.rpc_request("getTextURLsToUnfurl", params)
        return response

    def unfurl_urls(self, urls: list[str]):
        params = [urls]
        response = self.rpc_request("unfurlURLs", params)
        return response
