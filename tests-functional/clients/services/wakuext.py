from typing import TypedDict
from clients.rpc import RpcClient
from clients.services.service import Service
from resources.enums import MessageContentType


class SendPinMessagePayload(TypedDict):
    chat_id: str
    message_id: str
    pinned: bool


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

    def accept_latest_contact_request_for_contact(self, request_id: str):
        params = [{"id": request_id}]
        response = self.rpc_request("acceptLatestContactRequestForContact", params)
        return response.json()

    def decline_contact_request(self, request_id: str):
        params = [{"id": request_id}]
        response = self.rpc_request("declineContactRequest", params)
        return response.json()

    def dismiss_latest_contact_request_for_contact(self, request_id: str):
        params = [{"id": request_id}]
        response = self.rpc_request("dismissLatestContactRequestForContact", params)
        return response.json()

    def get_latest_contact_request_for_contact(self, request_id: str):
        params = [request_id]
        response = self.rpc_request("getLatestContactRequestForContact", params)
        return response.json()

    def retract_contact_request(self, request_id: str):
        params = [{"id": request_id}]
        response = self.rpc_request("retractContactRequest", params)
        return response.json()

    def remove_contact(self, request_id: str):
        params = [request_id]
        response = self.rpc_request("removeContact", params)
        return response.json()

    def set_contact_local_nickname(self, request_id: str, nickname: str):
        params = [{"id": request_id, "nickname": nickname}]
        response = self.rpc_request("setContactLocalNickname", params)
        return response.json()

    def get_contacts(self):
        response = self.rpc_request("contacts")
        return response.json()

    def add_contact(self, contact_id: str, displayName: str):
        params = [{"id": contact_id, "nickname": "fake_nickname", "displayName": displayName, "ensName": ""}]
        response = self.rpc_request("addContact", params)
        return response.json()

    def send_one_to_one_message(self, contact_id: str, message: str):
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

    def leave_community(self, community_id):
        params = [community_id]
        response = self.rpc_request("leaveCommunity", params)
        return response.json()

    def set_light_client(self, enabled=True):
        params = [{"enabled": enabled}]
        response = self.rpc_request("setLightClient", params)
        return response.json()

    def peers(self, enable_logging=True):
        params = []
        response = self.rpc_request("peers", params, enable_logging=enable_logging)
        return response.json()

    def chat_messages(self, chat_id: str, cursor="", limit=10):
        params = [chat_id, cursor, limit]
        response = self.rpc_request("chatMessages", params)
        return response.json()

    def message_by_message_id(self, message_id: str, skip_validation=False):
        params = [message_id]
        response = self.rpc_request("messageByMessageID", params, skip_validation=skip_validation)
        return response.json()

    def all_messages_from_chat_which_match_term(self, chat_id: str, searchTerm: str, caseSensitive: bool):
        params = [chat_id, searchTerm, caseSensitive]
        response = self.rpc_request("allMessagesFromChatWhichMatchTerm", params)
        return response.json()

    def send_pin_message(self, message: SendPinMessagePayload):
        params = [message]
        response = self.rpc_request("sendPinMessage", params)
        return response.json()

    def chat_pinned_messages(self, chat_id: str, cursor="", limit=10):
        params = [chat_id, cursor, limit]
        response = self.rpc_request("chatPinnedMessages", params)
        return response.json()

    def set_user_status(self, new_status: int, custom_text=""):
        params = [new_status, custom_text]
        response = self.rpc_request("setUserStatus", params)
        return response.json()

    def status_updates(self):
        params = []
        response = self.rpc_request("statusUpdates", params)
        return response.json()

    def edit_message(self, message_id: str, new_text: str):
        params = [{"id": message_id, "text": new_text}]
        response = self.rpc_request("editMessage", params)
        return response.json()

    def delete_message(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("deleteMessage", params)
        return response.json()

    def delete_messages_by_chat_id(self, chat_id: str):
        params = [chat_id]
        response = self.rpc_request("deleteMessagesByChatID", params)
        return response.json()

    def delete_message_and_send(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("deleteMessageAndSend", params)
        return response.json()

    def delete_message_for_me_and_sync(self, local_chat_id: str, message_id: str):
        params = [local_chat_id, message_id]
        response = self.rpc_request("deleteMessageForMeAndSync", params)
        return response.json()

    def mark_message_as_unread(self, chat_id: str, message_id: str):
        params = [chat_id, message_id]
        response = self.rpc_request("markMessageAsUnread", params)
        return response.json()

    def first_unseen_message_id(self, chat_id: str):
        params = [chat_id]
        response = self.rpc_request("firstUnseenMessageID", params)
        return response.json()

    def update_message_outgoing_status(self, message_id: str, new_status: str):
        params = [message_id, new_status]
        response = self.rpc_request("updateMessageOutgoingStatus", params)
        return response.json()
