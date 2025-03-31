from typing import TypedDict
from clients.rpc import RpcClient
from clients.services.service import Service
from resources.enums import MessageContentType


class SendPinMessagePayload(TypedDict):
    chat_id: str
    message_id: str
    pinned: bool


class SendChatMessagePayload(TypedDict):
    chat_id: str
    text: str
    content_type: int


class WakuextService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "wakuext")

    def start_messenger(self):
        response = self.rpc_request("startMessenger")
        json_response = response.json()

        if "error" in json_response:
            assert json_response["error"]["code"] == -32000
            assert json_response["error"]["message"] == "messenger already started"
            return

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

    def send_chat_message(self, chat_id, message, content_type=MessageContentType.TEXT_PLAIN.value):
        params = [{"chatId": chat_id, "text": message, "contentType": content_type}]
        response = self.rpc_request("sendChatMessage", params)
        return response.json()

    def send_chat_messages(self, messages: list[SendChatMessagePayload]):
        params = [[{"chatId": m["chat_id"], "text": m["text"], "contentType": m["content_type"]} for m in messages]]
        response = self.rpc_request("sendChatMessages", params)
        return response.json()

    def resend_chat_message(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("reSendChatMessage", params)
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

    def all_messages_from_chats_and_communities_which_match_term(
        self, community_ids: list[str], chat_ids: list[str], searchTerm: str, caseSensitive: bool
    ):
        params = [community_ids, chat_ids, searchTerm, caseSensitive]
        response = self.rpc_request("allMessagesFromChatsAndCommunitiesWhichMatchTerm", params)
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

    def request_transaction(self, chat_id: str, value: str, contract: str, address: str):
        params = [chat_id, value, contract, address]
        response = self.rpc_request("requestTransaction", params)
        return response.json()

    def decline_request_transaction(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("declineRequestTransaction", params)
        return response.json()

    def accept_request_transaction(self, transactionHash: str, message_id: str, signature: str):
        params = [transactionHash, message_id, signature]
        response = self.rpc_request("acceptRequestTransaction", params)
        return response.json()

    def request_address_for_transaction(self, chat_id: str, address_from: str, value: str, contract: str):
        params = [chat_id, address_from, value, contract]
        response = self.rpc_request("requestAddressForTransaction", params)
        return response.json()

    def decline_request_address_for_transaction(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("declineRequestAddressForTransaction", params)
        return response.json()

    def accept_request_address_for_transaction(self, message_id: str, address: str):
        params = [message_id, address]
        response = self.rpc_request("acceptRequestAddressForTransaction", params)
        return response.json()

    def send_transaction(self, chat_id: str, value: str, contract: str, transactionHash: str, signature: str):
        params = [chat_id, value, contract, transactionHash, signature]
        response = self.rpc_request("sendTransaction", params)
        return response.json()

    def chats(self):
        params = []
        response = self.rpc_request("chats", params)
        return response.json()

    def chat(self, chat_id: str):
        params = [chat_id]
        response = self.rpc_request("chat", params)
        return response.json()

    def chats_preview(self, filter_type: int):
        params = [filter_type]
        response = self.rpc_request("chatsPreview", params)
        return response.json()

    def active_chats(self):
        params = []
        response = self.rpc_request("activeChats", params)
        return response.json()

    def mute_chat(self, chat_id: str):
        params = [chat_id]
        response = self.rpc_request("muteChat", params)
        return response.json()

    def mute_chat_v2(self, chat_id: str, muted_type: int):
        params = [{"ChatId": chat_id, "MutedType": muted_type}]
        response = self.rpc_request("muteChatV2", params)
        return response.json()

    def unmute_chat(self, chat_id: str):
        params = [chat_id]
        response = self.rpc_request("unmuteChat", params)
        return response.json()

    def clear_history(self, chat_id: str):
        params = [{"id": chat_id}]
        response = self.rpc_request("clearHistory", params)
        return response.json()

    def deactivate_chat(self, chat_id: str, preserve_history: bool):
        params = [{"id": chat_id, "preserveHistory": preserve_history}]
        response = self.rpc_request("deactivateChat", params)
        return response.json()

    def save_chat(self, chat_id: str, active=True):
        params = [{"id": chat_id, "active": active}]
        response = self.rpc_request("saveChat", params)
        return response.json()

    def create_one_to_one_chat(self, chat_id: str, ens_name: str):
        params = [{"id": chat_id, "ensName": ens_name}]
        response = self.rpc_request("createOneToOneChat", params)
        return response.json()
