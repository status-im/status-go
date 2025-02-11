from clients.rpc import RpcClient
from clients.services.service import Service
from resources.enums import MessageContentType


class WakuextService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "wakuext")

    def send_contact_request(self, contact_id: str, message: str):
        params = [{"id": contact_id, "message": message}]
        response = self.rpc_request("sendContactRequest", params)
        return response.json()

    def accept_contact_request(self, request_id: str):
        params = [{"id": request_id}]
        response = self.rpc_request("acceptContactRequest", params)
        return response.json()

    def get_contacts(self):
        response = self.rpc_request("contacts")
        return response.json()

    def send_message(self, contact_id: str, message: str):
        params = [{"id": contact_id, "message": message}]
        response = self.rpc_request("sendOneToOneMessage", params)
        return response.json()

    def start_messenger(self):
        response = self.rpc_request("startMessenger")
        json_response = response.json()

        if "error" in json_response:
            assert json_response["error"]["code"] == -32000
            assert json_response["error"]["message"] == "messenger already started"
            return

    def create_group_chat_with_members(self, pubkey_list: list, group_chat_name: str):
        params = [None, group_chat_name, pubkey_list]
        response = self.rpc_request("createGroupChatWithMembers", params)
        return response.json()

    def send_group_chat_message(self, group_id: str, message: str):
        params = [{"id": group_id, "message": message}]
        response = self.rpc_request("sendGroupChatMessage", params)
        return response.json()

    def create_community(self, name, color="#ffffff", membership=3):
        params = [{"membership": membership, "name": name, "color": color, "description": name}]
        response = self.rpc_request("createCommunity", params)
        return response.json()

    def fetch_community(self, community_key):
        params = [{"communityKey": community_key, "waitForResponse": True, "tryDatabase": True}]
        response = self.rpc_request("fetchCommunity", params)
        return response.json()

    def request_to_join_community(self, community_id, address="fakeaddress"):
        params = [{"communityId": community_id, "addressesToReveal": [address], "airdropAddress": address}]
        response = self.rpc_request("requestToJoinCommunity", params)
        return response.json()

    def accept_request_to_join_community(self, request_to_join_id):
        params = [{"id": request_to_join_id}]
        response = self.rpc_request("acceptRequestToJoinCommunity", params)
        return response.json()

    def send_community_chat_message(self, chat_id, message, content_type=MessageContentType.TEXT_PLAIN.value):
        params = [{"chatId": chat_id, "text": message, "contentType": content_type}]
        response = self.rpc_request("sendChatMessage", params)
        return response.json()

    def set_light_client(self, enabled=True):
        params = [{"enabled": enabled}]
        response = self.rpc_request("setLightClient", params)
        return response.json()
