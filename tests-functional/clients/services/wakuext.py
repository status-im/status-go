from typing import TypedDict, Union
from clients.rpc import RpcClient
from clients.services.service import Service
from resources.enums import MessageContentType
from enum import Enum


class PushNotificationRegistrationTokenType(Enum):
    UNKNOWN = 0
    APN_TOKEN = 1
    FIREBASE_TOKEN = 2


class ActivityCenterNotificationType(Enum):
    NOTIFICATION_NO_TYPE = 0
    NOTIFICATION_TYPE_NEW_ONE_TO_ONE = 1
    NOTIFICATION_TYPE_NEW_PRIVATE_GROUP_CHAT = 2
    NOTIFICATION_TYPE_MENTION = 3
    NOTIFICATION_TYPE_REPLY = 4
    NOTIFICATION_TYPE_CONTACT_REQUEST = 5
    NOTIFICATION_TYPE_COMMUNITY_INVITATION = 6
    NOTIFICATION_TYPE_COMMUNITY_REQUEST = 7
    NOTIFICATION_TYPE_COMMUNITY_MEMBERSHIP_REQUEST = 8
    NOTIFICATION_TYPE_COMMUNITY_KICKED = 9
    NOTIFICATION_TYPE_CONTACT_VERIFICATION = 10
    NOTIFICATION_TYPE_CONTACT_REMOVED = 11
    NOTIFICATION_TYPE_NEW_KEYPAIR_ADDED_TO_PAIRED_DEVICE = 12
    NOTIFICATION_TYPE_OWNER_TOKEN_RECEIVED = 13
    NOTIFICATION_TYPE_OWNERSHIP_RECEIVED = 14
    NOTIFICATION_TYPE_OWNERSHIP_LOST = 15
    NOTIFICATION_TYPE_SET_SIGNER_FAILED = 16
    NOTIFICATION_TYPE_SET_SIGNER_DECLINED = 17
    NOTIFICATION_TYPE_SHARE_ACCOUNTS = 18
    NOTIFICATION_TYPE_COMMUNITY_TOKEN_RECEIVED = 19
    NOTIFICATION_TYPE_FIRST_COMMUNITY_TOKEN_RECEIVED = 20
    NOTIFICATION_TYPE_COMMUNITY_BANNED = 21
    NOTIFICATION_TYPE_COMMUNITY_UNBANNED = 22
    NOTIFICATION_TYPE_NEW_INSTALLATION_RECEIVED = 23
    NOTIFICATION_TYPE_NEW_INSTALLATION_CREATED = 24
    NOTIFICATION_TYPE_BACKUP_SYNCING_FETCHING = 25
    NOTIFICATION_TYPE_BACKUP_SYNCING_SUCCESS = 26
    NOTIFICATION_TYPE_BACKUP_SYNCING_PARTIAL_FAILURE = 27
    NOTIFICATION_TYPE_BACKUP_SYNCING_FAILURE = 28
    NOTIFICATION_TYPE_NEWS = 29


class ActivityCenterMembershipStatus(Enum):
    IDLE = 0
    PENDING = 1
    ACCEPTED = 2
    DECLINED = 3
    ACCEPTED_PENDING = 4
    DECLINED_PENDING = 5
    OWNERSHIP_CHANGED = 6


class ActivityCenterQueryParamsRead(Enum):
    READ = 1
    UNREAD = 2
    ALL = 3


class ContactRequestState(Enum):
    NONE = 0
    MUTUAL = 1
    SENT = 2
    RECEIVED = 3
    DISMISSED = 4


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

    def start_messenger(self, **kwargs):
        response = self.rpc_request("startMessenger", **kwargs)
        json_response = response.json()

        if "error" in json_response:
            assert json_response["error"]["code"] == -32000
            assert json_response["error"]["message"] == "messenger already started"
            return

    def send_contact_request(self, contact_id: str, message: str, **kwargs):
        params = [{"id": contact_id, "message": message}]
        response = self.rpc_request("sendContactRequest", params, **kwargs)
        return response.json()

    def accept_contact_request(self, request_id: str, **kwargs):
        params = [{"id": request_id}]
        response = self.rpc_request("acceptContactRequest", params, **kwargs)
        return response.json()

    def accept_latest_contact_request_for_contact(self, request_id: str, **kwargs):
        params = [{"id": request_id}]
        response = self.rpc_request("acceptLatestContactRequestForContact", params, **kwargs)
        return response.json()

    def decline_contact_request(self, request_id: str, **kwargs):
        params = [{"id": request_id}]
        response = self.rpc_request("declineContactRequest", params, **kwargs)
        return response.json()

    def dismiss_latest_contact_request_for_contact(self, request_id: str, **kwargs):
        params = [{"id": request_id}]
        response = self.rpc_request("dismissLatestContactRequestForContact", params, **kwargs)
        return response.json()

    def get_latest_contact_request_for_contact(self, request_id: str, **kwargs):
        params = [request_id]
        response = self.rpc_request("getLatestContactRequestForContact", params, **kwargs)
        return response.json()

    def retract_contact_request(self, request_id: str, **kwargs):
        params = [{"id": request_id}]
        response = self.rpc_request("retractContactRequest", params, **kwargs)
        return response.json()

    def remove_contact(self, request_id: str, **kwargs):
        params = [request_id]
        response = self.rpc_request("removeContact", params, **kwargs)
        return response.json()

    def set_contact_local_nickname(self, request_id: str, nickname: str, **kwargs):
        params = [{"id": request_id, "nickname": nickname}]
        response = self.rpc_request("setContactLocalNickname", params, **kwargs)
        return response.json()

    def get_contacts(self, **kwargs):
        response = self.rpc_request("contacts", **kwargs)
        return response.json()

    def add_contact(self, contact_id: str, displayName: str, **kwargs):
        params = [{"id": contact_id, "nickname": "fake_nickname", "displayName": displayName, "ensName": ""}]
        response = self.rpc_request("addContact", params, **kwargs)
        return response.json()

    def send_one_to_one_message(self, contact_id: str, message: str, **kwargs):
        params = [{"id": contact_id, "message": message}]
        response = self.rpc_request("sendOneToOneMessage", params, **kwargs)
        return response.json()

    def create_group_chat_with_members(self, pubkey_list: list, group_chat_name: str, **kwargs):
        params = [None, group_chat_name, pubkey_list]
        response = self.rpc_request("createGroupChatWithMembers", params, **kwargs)
        return response.json()

    def send_group_chat_message(self, group_id: str, message: str, **kwargs):
        params = [{"id": group_id, "message": message}]
        response = self.rpc_request("sendGroupChatMessage", params, **kwargs)
        return response.json()

    def leave_group_chat(self, chat_id: str, remove: bool, **kwargs):
        params = [None, chat_id, remove]
        response = self.rpc_request("leaveGroupChat", params, **kwargs)
        return response.json()

    def create_group_chat_from_invitation(self, name: str, chat_id: str, admin_pk: str, **kwargs):
        params = [name, chat_id, admin_pk]
        response = self.rpc_request("createGroupChatFromInvitation", params, **kwargs)
        return response.json()

    def add_members_to_group_chat(self, chat_id: str, members: list, **kwargs):
        params = [None, chat_id, members]
        response = self.rpc_request("addMembersToGroupChat", params, **kwargs)
        return response.json()

    def remove_member_from_group_chat(self, chat_id: str, member: str, **kwargs):
        params = [None, chat_id, member]
        response = self.rpc_request("removeMemberFromGroupChat", params, **kwargs)
        return response.json()

    def remove_members_from_group_chat(self, chat_id: str, members: list, **kwargs):
        params = [None, chat_id, members]
        response = self.rpc_request("removeMembersFromGroupChat", params, **kwargs)
        return response.json()

    def confirm_joining_group(self, chat_id: str, **kwargs):
        params = [chat_id]
        response = self.rpc_request("confirmJoiningGroup", params, **kwargs)
        return response.json()

    def change_group_chat_name(self, chat_id: str, name: str, **kwargs):
        params = [None, chat_id, name]
        response = self.rpc_request("changeGroupChatName", params, **kwargs)
        return response.json()

    def send_group_chat_invitation_request(self, chat_id: str, admin_pk: str, message: str, **kwargs):
        params = [None, chat_id, admin_pk, message]
        response = self.rpc_request("sendGroupChatInvitationRequest", params, **kwargs)
        return response.json()

    def get_group_chat_invitations(self, **kwargs):
        response = self.rpc_request("getGroupChatInvitations", **kwargs)
        return response.json()

    def send_group_chat_invitation_rejection(self, invitation_request_id: str, **kwargs):
        params = [None, invitation_request_id]
        response = self.rpc_request("sendGroupChatInvitationRejection", params, **kwargs)
        return response.json()

    def create_community(self, name, color="#ffffff", membership=3, **kwargs):
        params = [{"membership": membership, "name": name, "color": color, "description": name}]
        response = self.rpc_request("createCommunity", params, **kwargs)
        return response.json()

    def fetch_community(self, community_key, **kwargs):
        params = [{"communityKey": community_key, "waitForResponse": True, "tryDatabase": True}]
        response = self.rpc_request("fetchCommunity", params, **kwargs)
        return response.json()

    def request_to_join_community(self, community_id, address="fakeaddress", **kwargs):
        params = [{"communityId": community_id, "addressesToReveal": [address], "airdropAddress": address}]
        response = self.rpc_request("requestToJoinCommunity", params, **kwargs)
        return response.json()

    def accept_request_to_join_community(self, request_to_join_id, **kwargs):
        params = [{"id": request_to_join_id}]
        response = self.rpc_request("acceptRequestToJoinCommunity", params, **kwargs)
        return response.json()

    def send_chat_message(self, chat_id, message, content_type=MessageContentType.TEXT_PLAIN.value, **kwargs):
        params = [{"chatId": chat_id, "text": message, "contentType": content_type}]
        response = self.rpc_request("sendChatMessage", params, **kwargs)
        return response.json()

    def send_chat_messages(self, messages: list[SendChatMessagePayload], **kwargs):
        params = [[{"chatId": m["chat_id"], "text": m["text"], "contentType": m["content_type"]} for m in messages]]
        response = self.rpc_request("sendChatMessages", params, **kwargs)
        return response.json()

    def resend_chat_message(self, message_id: str, **kwargs):
        params = [message_id]
        response = self.rpc_request("reSendChatMessage", params, **kwargs)
        return response.json()

    def leave_community(self, community_id, **kwargs):
        params = [community_id]
        response = self.rpc_request("leaveCommunity", params, **kwargs)
        return response.json()

    def set_light_client(self, enabled=True, **kwargs):
        params = [{"enabled": enabled}]
        response = self.rpc_request("setLightClient", params, **kwargs)
        return response.json()

    def peers(self, **kwargs):
        params = []
        response = self.rpc_request("peers", params, **kwargs)
        return response.json()

    def chat_messages(self, chat_id: str, cursor="", limit=10, **kwargs):
        params = [chat_id, cursor, limit]
        response = self.rpc_request("chatMessages", params, **kwargs)
        return response.json()

    def message_by_message_id(self, message_id: str, **kwargs):
        params = [message_id]
        response = self.rpc_request("messageByMessageID", params, **kwargs)
        return response.json()

    def all_messages_from_chat_which_match_term(self, chat_id: str, searchTerm: str, caseSensitive: bool, **kwargs):
        params = [chat_id, searchTerm, caseSensitive]
        response = self.rpc_request("allMessagesFromChatWhichMatchTerm", params, **kwargs)
        return response.json()

    def all_messages_from_chats_and_communities_which_match_term(
        self, community_ids: list[str], chat_ids: list[str], searchTerm: str, caseSensitive: bool, **kwargs
    ):
        params = [community_ids, chat_ids, searchTerm, caseSensitive]
        response = self.rpc_request("allMessagesFromChatsAndCommunitiesWhichMatchTerm", params, **kwargs)
        return response.json()

    def send_pin_message(self, message: SendPinMessagePayload, **kwargs):
        params = [message]
        response = self.rpc_request("sendPinMessage", params, **kwargs)
        return response.json()

    def chat_pinned_messages(self, chat_id: str, cursor="", limit=10, **kwargs):
        params = [chat_id, cursor, limit]
        response = self.rpc_request("chatPinnedMessages", params, **kwargs)
        return response.json()

    def set_user_status(self, new_status: int, custom_text="", **kwargs):
        params = [new_status, custom_text]
        response = self.rpc_request("setUserStatus", params, **kwargs)
        return response.json()

    def status_updates(self, **kwargs):
        params = []
        response = self.rpc_request("statusUpdates", params, **kwargs)
        return response.json()

    def edit_message(self, message_id: str, new_text: str, **kwargs):
        params = [{"id": message_id, "text": new_text}]
        response = self.rpc_request("editMessage", params, **kwargs)
        return response.json()

    def delete_message(self, message_id: str, **kwargs):
        params = [message_id]
        response = self.rpc_request("deleteMessage", params, **kwargs)
        return response.json()

    def delete_messages_by_chat_id(self, chat_id: str, **kwargs):
        params = [chat_id]
        response = self.rpc_request("deleteMessagesByChatID", params, **kwargs)
        return response.json()

    def delete_message_and_send(self, message_id: str, **kwargs):
        params = [message_id]
        response = self.rpc_request("deleteMessageAndSend", params, **kwargs)
        return response.json()

    def delete_message_for_me_and_sync(self, local_chat_id: str, message_id: str, **kwargs):
        params = [local_chat_id, message_id]
        response = self.rpc_request("deleteMessageForMeAndSync", params, **kwargs)
        return response.json()

    def mark_message_as_unread(self, chat_id: str, message_id: str, **kwargs):
        params = [chat_id, message_id]
        response = self.rpc_request("markMessageAsUnread", params, **kwargs)
        return response.json()

    def first_unseen_message_id(self, chat_id: str, **kwargs):
        params = [chat_id]
        response = self.rpc_request("firstUnseenMessageID", params, **kwargs)
        return response.json()

    def update_message_outgoing_status(self, message_id: str, new_status: str, **kwargs):
        params = [message_id, new_status]
        response = self.rpc_request("updateMessageOutgoingStatus", params, **kwargs)
        return response.json()

    def request_transaction(self, chat_id: str, value: str, contract: str, address: str, **kwargs):
        params = [chat_id, value, contract, address]
        response = self.rpc_request("requestTransaction", params, **kwargs)
        return response.json()

    def decline_request_transaction(self, message_id: str, **kwargs):
        params = [message_id]
        response = self.rpc_request("declineRequestTransaction", params, **kwargs)
        return response.json()

    def accept_request_transaction(self, transactionHash: str, message_id: str, signature: str, **kwargs):
        params = [transactionHash, message_id, signature]
        response = self.rpc_request("acceptRequestTransaction", params, **kwargs)
        return response.json()

    def request_address_for_transaction(self, chat_id: str, address_from: str, value: str, contract: str, **kwargs):
        params = [chat_id, address_from, value, contract]
        response = self.rpc_request("requestAddressForTransaction", params, **kwargs)
        return response.json()

    def decline_request_address_for_transaction(self, message_id: str, **kwargs):
        params = [message_id]
        response = self.rpc_request("declineRequestAddressForTransaction", params, **kwargs)
        return response.json()

    def accept_request_address_for_transaction(self, message_id: str, address: str, **kwargs):
        params = [message_id, address]
        response = self.rpc_request("acceptRequestAddressForTransaction", params, **kwargs)
        return response.json()

    def send_transaction(self, chat_id: str, value: str, contract: str, transactionHash: str, signature: str, **kwargs):
        params = [chat_id, value, contract, transactionHash, signature]
        response = self.rpc_request("sendTransaction", params, **kwargs)
        return response.json()

    def chats(self, **kwargs):
        params = []
        response = self.rpc_request("chats", params, **kwargs)
        return response.json()

    def chat(self, chat_id: str, **kwargs):
        params = [chat_id]
        response = self.rpc_request("chat", params, **kwargs)
        return response.json()

    def chats_preview(self, filter_type: int, **kwargs):
        params = [filter_type]
        response = self.rpc_request("chatsPreview", params, **kwargs)
        return response.json()

    def active_chats(self, **kwargs):
        params = []
        response = self.rpc_request("activeChats", params, **kwargs)
        return response.json()

    def mute_chat(self, chat_id: str, **kwargs):
        params = [chat_id]
        response = self.rpc_request("muteChat", params, **kwargs)
        return response.json()

    def mute_chat_v2(self, chat_id: str, muted_type: int, **kwargs):
        params = [{"ChatId": chat_id, "MutedType": muted_type}]
        response = self.rpc_request("muteChatV2", params, **kwargs)
        return response.json()

    def unmute_chat(self, chat_id: str, **kwargs):
        params = [chat_id]
        response = self.rpc_request("unmuteChat", params, **kwargs)
        return response.json()

    def clear_history(self, chat_id: str, **kwargs):
        params = [{"id": chat_id}]
        response = self.rpc_request("clearHistory", params, **kwargs)
        return response.json()

    def deactivate_chat(self, chat_id: str, preserve_history: bool, **kwargs):
        params = [{"id": chat_id, "preserveHistory": preserve_history}]
        response = self.rpc_request("deactivateChat", params, **kwargs)
        return response.json()

    def save_chat(self, chat_id: str, active=True, **kwargs):
        params = [{"id": chat_id, "active": active}]
        response = self.rpc_request("saveChat", params, **kwargs)
        return response.json()

    def create_one_to_one_chat(self, chat_id: str, ens_name: str, **kwargs):
        params = [{"id": chat_id, "ensName": ens_name}]
        response = self.rpc_request("createOneToOneChat", params, **kwargs)
        return response.json()

    def register_for_push_notifications(self, device_token: str, apnTopic: str, tokenType: PushNotificationRegistrationTokenType, **kwargs):
        params = [device_token, apnTopic, tokenType.value]
        response = self.rpc_request("registerForPushNotifications", params, **kwargs)
        return response.json()

    def get_activity_center_notifications(
        self,
        activity_types: list = list(ActivityCenterNotificationType),
        read_type: Union[ActivityCenterQueryParamsRead, None] = None,
        cursor: str = "",
        limit: int = 20,
        **kwargs,
    ):
        params = {
            "activityTypes": [item.value for item in activity_types],
            "cursor": cursor,
            "limit": limit,
        }
        if read_type is not None:
            params["readType"] = read_type.value

        response = self.rpc_request(method="activityCenterNotifications", params=[params], **kwargs)
        return response.json()
