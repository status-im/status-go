from clients.rpc import RpcClient
from clients.services.service import Service


class SharedURLsService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "sharedurls")

    def share_community_url_with_chat_key(self, community_id: str):
        params = [community_id]
        response = self.rpc_request("shareCommunityURLWithChatKey", params)
        return response

    def share_community_url_with_data(self, community_id: str):
        params = [community_id]
        response = self.rpc_request("shareCommunityURLWithData", params)
        return response

    def share_community_channel_url_with_chat_key(self, community_id: str, channel_id: str):
        params = [community_id, channel_id]
        response = self.rpc_request("shareCommunityChannelURLWithChatKey", params)
        return response

    def share_community_channel_url_with_data(self, community_id: str, channel_id: str):
        params = [community_id, channel_id]
        response = self.rpc_request("shareCommunityChannelURLWithData", params)
        return response

    def share_user_url_with_ens(self, pub_key: str):
        params = [pub_key]
        response = self.rpc_request("shareUserURLWithENS", params)
        return response

    def share_user_url_with_chat_key(self, pub_key: str):
        params = [pub_key]
        response = self.rpc_request("shareUserURLWithChatKey", params)
        return response

    def share_user_url_with_data(self, pub_key: str):
        params = [pub_key]
        response = self.rpc_request("shareUserURLWithData", params)
        return response

    def parse_shared_url(self, url: str):
        params = [url]
        response = self.rpc_request("parseSharedURL", params)
        return response
