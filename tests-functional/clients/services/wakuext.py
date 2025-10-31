from enum import Enum
from typing import TypedDict, Union

from clients.rpc import RpcClient
from clients.services.service import Service
from resources.enums import MessageContentType
from utils.image_utils import ImageCropRect


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


class CommunityPermissionsAccess(Enum):
    UNKNOWN = 0
    AUTO_ACCEPT = 1
    MANUAL_ACCEPT = 3


class Error(Exception):
    def __init__(self, message):
        self.message = message


class WakuextService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "wakuext")

    def start_messenger(self):
        self.rpc_request("startMessenger")

    def send_contact_request(self, contact_id: str, message: str):
        params = [{"id": contact_id, "message": message}]
        response = self.rpc_request("sendContactRequest", params)
        return response

    def accept_contact_request(self, request_id: str):
        params = [{"id": request_id}]
        response = self.rpc_request("acceptContactRequest", params)
        return response

    def accept_latest_contact_request_for_contact(self, request_id: str):
        params = [{"id": request_id}]
        response = self.rpc_request("acceptLatestContactRequestForContact", params)
        return response

    def decline_contact_request(self, request_id: str):
        params = [{"id": request_id}]
        response = self.rpc_request("declineContactRequest", params)
        return response

    def dismiss_latest_contact_request_for_contact(self, request_id: str):
        params = [{"id": request_id}]
        response = self.rpc_request("dismissLatestContactRequestForContact", params)
        return response

    def get_latest_contact_request_for_contact(self, request_id: str):
        params = [request_id]
        response = self.rpc_request("getLatestContactRequestForContact", params)
        return response

    def retract_contact_request(self, request_id: str):
        params = [{"id": request_id}]
        response = self.rpc_request("retractContactRequest", params)
        return response

    def remove_contact(self, request_id: str):
        params = [request_id]
        response = self.rpc_request("removeContact", params)
        return response

    def set_contact_local_nickname(self, request_id: str, nickname: str):
        params = [{"id": request_id, "nickname": nickname}]
        response = self.rpc_request("setContactLocalNickname", params)
        return response

    def get_contacts(self):
        response = self.rpc_request("contacts")
        return response

    def get_contact_by_id(self, id: str):
        params = [id]
        response = self.rpc_request("getContactByID", params)
        return response

    def add_contact(self, contact_id: str, displayName: str):
        params = [{"id": contact_id, "nickname": "fake_nickname", "displayName": displayName, "ensName": ""}]
        response = self.rpc_request("addContact", params)
        return response

    def send_one_to_one_message(self, contact_id: str, message: str):
        params = [{"id": contact_id, "message": message}]
        response = self.rpc_request("sendOneToOneMessage", params)
        return response

    def create_group_chat_with_members(self, pubkey_list: list, group_chat_name: str):
        params = [group_chat_name, pubkey_list]
        response = self.rpc_request("createGroupChatWithMembers", params)
        return response

    def send_group_chat_message(self, group_id: str, message: str):
        params = [{"id": group_id, "message": message}]
        response = self.rpc_request("sendGroupChatMessage", params)
        return response

    def leave_group_chat(self, chat_id: str, remove: bool):
        params = [chat_id, remove]
        response = self.rpc_request("leaveGroupChat", params)
        return response

    def create_group_chat_from_invitation(self, name: str, chat_id: str, admin_pk: str):
        params = [name, chat_id, admin_pk]
        response = self.rpc_request("createGroupChatFromInvitation", params)
        return response

    def add_members_to_group_chat(self, chat_id: str, members: list):
        params = [chat_id, members]
        response = self.rpc_request("addMembersToGroupChat", params)
        return response

    def remove_member_from_group_chat(self, chat_id: str, member: str):
        params = [chat_id, member]
        response = self.rpc_request("removeMemberFromGroupChat", params)
        return response

    def remove_members_from_group_chat(self, chat_id: str, members: list):
        params = [chat_id, members]
        response = self.rpc_request("removeMembersFromGroupChat", params)
        return response

    def confirm_joining_group(self, chat_id: str):
        params = [chat_id]
        response = self.rpc_request("confirmJoiningGroup", params)
        return response

    def change_group_chat_name(self, chat_id: str, name: str):
        params = [chat_id, name]
        response = self.rpc_request("changeGroupChatName", params)
        return response

    def send_group_chat_invitation_request(self, chat_id: str, admin_pk: str, message: str):
        params = [chat_id, admin_pk, message]
        response = self.rpc_request("sendGroupChatInvitationRequest", params)
        return response

    def get_group_chat_invitations(self):
        response = self.rpc_request("getGroupChatInvitations")
        return response

    def send_group_chat_invitation_rejection(self, invitation_request_id: str):
        params = [invitation_request_id]
        response = self.rpc_request("sendGroupChatInvitationRejection", params)
        return response

    def create_community(
        self,
        name,
        description,
        color="#ffffff",
        membership: CommunityPermissionsAccess = CommunityPermissionsAccess.AUTO_ACCEPT,
        image="",
        image_rect=ImageCropRect(),
    ):
        params = {
            "membership": membership.value,
            "name": name,
            "color": color,
            "description": description,
            "image": image,
            "imageAx": image_rect.ax,
            "imageAy": image_rect.ay,
            "imageBx": image_rect.bx,
            "imageBy": image_rect.by,
        }
        response = self.rpc_request("createCommunity", [params])
        return response

    def edit_community(
        self,
        community_id,
        name,
        color="#ffffff",
        membership: CommunityPermissionsAccess = CommunityPermissionsAccess.AUTO_ACCEPT,
        description="",
        image="",
        image_rect=ImageCropRect(),
    ):
        params = {
            "CommunityID": community_id,
            "membership": membership.value,
            "name": name,
            "color": color,
            "description": description,
            "image": image,
            "imageAx": image_rect.ax,
            "imageAy": image_rect.ay,
            "imageBx": image_rect.bx,
            "imageBy": image_rect.by,
        }
        response = self.rpc_request("editCommunity", [params])
        return response

    def fetch_community(self, community_key):
        params = [{"communityKey": community_key, "waitForResponse": True, "tryDatabase": True}]
        response = self.rpc_request("fetchCommunity", params)
        return response

    def request_to_join_community(self, community_id, address="fakeaddress"):
        params = [{"communityId": community_id, "addressesToReveal": [address], "airdropAddress": address}]
        response = self.rpc_request("requestToJoinCommunity", params)
        return response

    def accept_request_to_join_community(self, request_to_join_id: str):
        params = [{"id": request_to_join_id}]
        response = self.rpc_request("acceptRequestToJoinCommunity", params)
        return response

    def cancel_request_to_join_community(self, request_to_join_id: str):
        params = [{"id": request_to_join_id}]
        response = self.rpc_request("cancelRequestToJoinCommunity", params)
        return response

    def decline_request_to_join_community(self, request_to_join_id: str):
        params = [{"id": request_to_join_id}]
        response = self.rpc_request("declineRequestToJoinCommunity", params)
        return response

    def canceled_requests_to_join_for_community(self, community_id: str):
        params = [community_id]
        response = self.rpc_request("canceledRequestsToJoinForCommunity", params)
        return response

    def pending_requests_to_join_for_community(self, community_id: str):
        params = [community_id]
        response = self.rpc_request("pendingRequestsToJoinForCommunity", params)
        return response

    def declined_requests_to_join_for_community(self, community_id: str):
        params = [community_id]
        response = self.rpc_request("declinedRequestsToJoinForCommunity", params)
        return response

    def latest_request_to_join_for_community(self, community_id: str):
        params = [community_id]
        response = self.rpc_request("latestRequestToJoinForCommunity", params)
        return response

    def my_pending_requests_to_join(self):
        params = []
        response = self.rpc_request("myPendingRequestsToJoin", params)
        return response

    def my_canceled_requests_to_join(self):
        params = []
        response = self.rpc_request("myCanceledRequestsToJoin", params)
        return response

    def check_and_delete_pending_request_to_join_community(self):
        params = []
        response = self.rpc_request("checkAndDeletePendingRequestToJoinCommunity", params)
        return response

    def all_non_approved_communities_requests_to_join(self):
        params = []
        response = self.rpc_request("allNonApprovedCommunitiesRequestsToJoin", params)
        return response

    def check_permissions_to_join_community(self, community_id: str):
        params = [{"communityId": community_id}]
        response = self.rpc_request("checkPermissionsToJoinCommunity", params)
        return response

    def generate_joining_community_requests_for_signing(self, member_pub_key: str, community_id: str, addresses_to_reveal: list):
        params = [member_pub_key, community_id, addresses_to_reveal]
        response = self.rpc_request("generateJoiningCommunityRequestsForSigning", params)
        return response

    def generate_edit_community_requests_for_signing(self, member_pub_key: str, community_id: str, addresses_to_reveal: list):
        params = [member_pub_key, community_id, addresses_to_reveal]
        response = self.rpc_request("generateEditCommunityRequestsForSigning", params)
        return response

    def send_chat_message(self, chat_id, message, content_type=MessageContentType.TEXT_PLAIN.value, responseTo: str = ""):
        params = [
            {
                "chatId": chat_id,
                "text": message,
                "contentType": content_type,
                "responseTo": responseTo,
            }
        ]
        response = self.rpc_request("sendChatMessage", params)
        return response

    def send_chat_messages(self, messages: list[SendChatMessagePayload]):
        params = [[{"chatId": m["chat_id"], "text": m["text"], "contentType": m["content_type"]} for m in messages]]
        response = self.rpc_request("sendChatMessages", params)
        return response

    def resend_chat_message(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("reSendChatMessage", params)
        return response

    def leave_community(self, community_id):
        params = [community_id]
        response = self.rpc_request("leaveCommunity", params)
        return response

    def set_light_client(self, enabled=True):
        params = [{"enabled": enabled}]
        response = self.rpc_request("setLightClient", params)
        return response

    def peers(self):
        params = []
        response = self.rpc_request("peers", params)
        return response

    def chat_messages(self, chat_id: str, cursor="", limit=10):
        params = [chat_id, cursor, limit]
        response = self.rpc_request("chatMessages", params)
        return response

    def message_by_message_id(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("messageByMessageID", params)
        return response

    def all_messages_from_chat_which_match_term(self, chat_id: str, searchTerm: str, caseSensitive: bool):
        params = [chat_id, searchTerm, caseSensitive]
        response = self.rpc_request("allMessagesFromChatWhichMatchTerm", params)
        return response

    def all_messages_from_chats_and_communities_which_match_term(
        self, community_ids: list[str], chat_ids: list[str], searchTerm: str, caseSensitive: bool
    ):
        params = [community_ids, chat_ids, searchTerm, caseSensitive]
        response = self.rpc_request("allMessagesFromChatsAndCommunitiesWhichMatchTerm", params)
        return response

    def send_pin_message(self, message: SendPinMessagePayload):
        params = [message]
        response = self.rpc_request("sendPinMessage", params)
        return response

    def chat_pinned_messages(self, chat_id: str, cursor="", limit=10):
        params = [chat_id, cursor, limit]
        response = self.rpc_request("chatPinnedMessages", params)
        return response

    def set_user_status(self, new_status: int, custom_text=""):
        params = [new_status, custom_text]
        response = self.rpc_request("setUserStatus", params)
        return response

    def set_bio(self, bio: str):
        params = [bio]
        response = self.rpc_request("setBio", params)
        return response

    def set_customization_color(self, color: str, key_uid: str):
        params = [{"customizationColor": color, "keyUid": key_uid}]
        response = self.rpc_request("setCustomizationColor", params)
        return response

    def set_syncing_on_mobile_network(self, enabled: bool):
        params = [{"enabled": enabled}]
        response = self.rpc_request("setSyncingOnMobileNetwork", params)
        return response

    def status_updates(self):
        params = []
        response = self.rpc_request("statusUpdates", params)
        return response

    def edit_message(self, message_id: str, new_text: str):
        params = [{"id": message_id, "text": new_text}]
        response = self.rpc_request("editMessage", params)
        return response

    def delete_message(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("deleteMessage", params)
        return response

    def delete_messages_by_chat_id(self, chat_id: str):
        params = [chat_id]
        response = self.rpc_request("deleteMessagesByChatID", params)
        return response

    def delete_message_and_send(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("deleteMessageAndSend", params)
        return response

    def delete_message_for_me_and_sync(self, local_chat_id: str, message_id: str):
        params = [local_chat_id, message_id]
        response = self.rpc_request("deleteMessageForMeAndSync", params)
        return response

    def mark_message_as_unread(self, chat_id: str, message_id: str):
        params = [chat_id, message_id]
        response = self.rpc_request("markMessageAsUnread", params)
        return response

    def first_unseen_message_id(self, chat_id: str):
        params = [chat_id]
        response = self.rpc_request("firstUnseenMessageID", params)
        return response

    def update_message_outgoing_status(self, message_id: str, new_status: str):
        params = [message_id, new_status]
        response = self.rpc_request("updateMessageOutgoingStatus", params)
        return response

    def chats(self):
        params = []
        response = self.rpc_request("chats", params)
        return response

    def chat(self, chat_id: str):
        params = [chat_id]
        response = self.rpc_request("chat", params)
        return response

    def chats_preview(self, filter_type: int):
        params = [filter_type]
        response = self.rpc_request("chatsPreview", params)
        return response

    def active_chats(self):
        params = []
        response = self.rpc_request("activeChats", params)
        return response

    def mute_chat(self, chat_id: str):
        params = [chat_id]
        response = self.rpc_request("muteChat", params)
        return response

    def mute_chat_v2(self, chat_id: str, muted_type: int):
        params = [{"ChatId": chat_id, "MutedType": muted_type}]
        response = self.rpc_request("muteChatV2", params)
        return response

    def unmute_chat(self, chat_id: str):
        params = [chat_id]
        response = self.rpc_request("unmuteChat", params)
        return response

    def clear_history(self, chat_id: str):
        params = [{"id": chat_id}]
        response = self.rpc_request("clearHistory", params)
        return response

    def deactivate_chat(self, chat_id: str, preserve_history: bool):
        params = [{"id": chat_id, "preserveHistory": preserve_history}]
        response = self.rpc_request("deactivateChat", params)
        return response

    def save_chat(self, chat_id: str, active=True):
        params = [{"id": chat_id, "active": active}]
        response = self.rpc_request("saveChat", params)
        return response

    def create_one_to_one_chat(self, chat_id: str, ens_name: str):
        params = [{"id": chat_id, "ensName": ens_name}]
        response = self.rpc_request("createOneToOneChat", params)
        return response

    def register_for_push_notifications(self, device_token: str, apnTopic: str, tokenType: PushNotificationRegistrationTokenType):
        params = [device_token, apnTopic, tokenType.value]
        response = self.rpc_request("registerForPushNotifications", params)
        return response

    def get_activity_center_notifications(
        self,
        activity_types: list = list(ActivityCenterNotificationType),
        read_type: Union[ActivityCenterQueryParamsRead, None] = None,
        cursor: str = "",
        limit: int = 20,
    ):
        params = {
            "activityTypes": [item.value for item in activity_types],
            "cursor": cursor,
            "limit": limit,
        }
        if read_type is not None:
            params["readType"] = read_type

        response = self.rpc_request(method="activityCenterNotifications", params=[params])
        return response

    def activity_center_notifications_count(
        self, activity_types: list = list(ActivityCenterNotificationType), read_type: Union[ActivityCenterQueryParamsRead, None] = None
    ):
        params = [{"activityTypes": activity_types, "readType": read_type}]
        response = self.rpc_request("activityCenterNotificationsCount", params)
        return response

    def has_unseen_activity_center_notifications(self):
        params = []
        response = self.rpc_request("hasUnseenActivityCenterNotifications", params)
        return response

    def mark_as_seen_activity_center_notifications(self):
        params = []
        response = self.rpc_request("markAsSeenActivityCenterNotifications", params)
        return response

    def mark_activity_center_notifications_read(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("markActivityCenterNotificationsRead", params)
        return response

    def mark_activity_center_notifications_unread(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("markActivityCenterNotificationsUnread", params)
        return response

    def mark_all_activity_center_notifications_read(self):
        params = []
        response = self.rpc_request("markAllActivityCenterNotificationsRead", params)
        return response

    def accept_activity_center_notifications(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("acceptActivityCenterNotifications", params)
        return response

    def dismiss_activity_center_notifications(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("dismissActivityCenterNotifications", params)
        return response

    def delete_activity_center_notifications(self, message_id: str):
        params = [message_id]
        response = self.rpc_request("deleteActivityCenterNotifications", params)
        return response

    def get_activity_center_state(self):
        params = []
        response = self.rpc_request("getActivityCenterState", params)
        return response

    def peer_id(self):
        params = []
        response = self.rpc_request("peerID", params)
        return response

    def send_emoji_reaction(self, receiver_chat_id: str, message_id: str, emoji_id: int):
        params = [receiver_chat_id, message_id, emoji_id]
        response = self.rpc_request(method="sendEmojiReaction", params=params)
        return response

    def send_emoji_reaction_v2(self, chat_id: str, message_id: str, emoji: str):
        params = [chat_id, message_id, emoji]
        response = self.rpc_request(method="sendEmojiReactionV2", params=params)
        return response

    def send_emoji_reaction_retraction(self, last_emoji_id: str):
        params = [last_emoji_id]
        response = self.rpc_request(method="sendEmojiReactionRetraction", params=params)
        return response

    def emoji_reactions_by_chat_id(self, sender_chat_id: str, limit: int):
        params = [sender_chat_id, None, limit]
        response = self.rpc_request(method="emojiReactionsByChatID", params=params)
        return response

    def emoji_reactions_by_chat_id_message_id(self, sender_chat_id: str, message_id: str):
        params = [sender_chat_id, message_id]
        response = self.rpc_request(method="emojiReactionsByChatIDMessageID", params=params)
        return response

    def get_saved_addresses(self, params=[]):
        response = self.rpc_request("getSavedAddresses", params)
        return response

    def get_saved_addresses_per_mode(self, is_test: bool):
        params = [is_test]
        response = self.rpc_request("getSavedAddressesPerMode", params)
        return response

    def upsert_saved_address(self, address: str, name: str, color_id: str, ens: str = "", chain_short_names: str = "", is_test: bool = False):
        params = [
            {
                "address": address,
                "name": name,
                "ens": ens,
                "colorId": color_id,
                "isTest": is_test,
                "chainShortNames": chain_short_names,
            },
        ]
        response = self.rpc_request("upsertSavedAddress", params)
        return response

    def delete_saved_address(self, address: str, is_test: bool):
        params = [address, is_test]
        response = self.rpc_request("deleteSavedAddress", params)
        return response

    def remaining_capacity_for_saved_addresses(self, is_test: bool):
        params = [is_test]
        response = self.rpc_request("remainingCapacityForSavedAddresses", params)
        return response

    def set_display_name(self, name: str):
        params = [name]
        response = self.rpc_request("setDisplayName", params)
        return response

    def set_profile_showcase_preferences(self, prefs: dict):
        response = self.rpc_request("setProfileShowcasePreferences", [prefs])
        return response

    def get_profile_showcase_preferences(self):
        params = []
        response = self.rpc_request("getProfileShowcasePreferences", params)
        return response

    def create_community_from_payload(self, community: dict):
        params = [community]
        response = self.rpc_request("createCommunity", params)
        return response

    def communities(self):
        params = []
        response = self.rpc_request("communities", params)
        return response

    def joined_communities(self):
        params = []
        response = self.rpc_request("joinedCommunities", params)
        return response

    def log_test(self):
        response = self.rpc_request("logTest")
        return response

    def create_community_chat(self, community_id: str, c: dict):
        params = [community_id, c]
        response = self.rpc_request("createCommunityChat", params)
        return response

    def edit_community_chat(self, community_id: str, chat_id: str, c: dict):
        params = [community_id, chat_id, c]
        response = self.rpc_request("editCommunityChat", params)
        return response

    def delete_community_chat(self, community_id: str, chat_id: str):
        params = [community_id, chat_id]
        response = self.rpc_request("deleteCommunityChat", params)
        return response

    def reorder_community_chat(self, community_id: str, chat_id: str, position: int):
        params = [{"communityId": community_id, "chatId": chat_id, "position": position}]
        response = self.rpc_request("reorderCommunityChat", params)
        return response

    def mute_community_chats(self, community_id: str, muted_type):
        params = [{"communityId": community_id, "mutedType": muted_type}]
        response = self.rpc_request("muteCommunityChats", params)
        return response

    def un_mute_community_chats(self, community_id: str):
        params = [community_id]
        response = self.rpc_request("unMuteCommunityChats", params)
        return response

    def accept_contact_verification_request(self, id: str, response: str):
        params = [id, response]
        response = self.rpc_request("acceptContactVerificationRequest", params)
        return response

    def decline_contact_verification_request(self, id: str):
        params = [id]
        response = self.rpc_request("declineContactVerificationRequest", params)
        return response

    def cancel_verification_request(self, id: str):
        params = [id]
        response = self.rpc_request("cancelVerificationRequest", params)
        return response

    def get_latest_verification_request_from(self, contact_id: str):
        params = [contact_id]
        response = self.rpc_request("getLatestVerificationRequestFrom", params)
        return response

    def send_contact_verification_request(self, contact_id: str, challenge: str):
        params = [contact_id, challenge]
        response = self.rpc_request("sendContactVerificationRequest", params)
        return response

    def get_received_verification_requests(self):
        params = []
        response = self.rpc_request("getReceivedVerificationRequests", params)
        return response

    def get_verification_request_sent_to(self, contact_id: str):
        params = [contact_id]
        response = self.rpc_request("getVerificationRequestSentTo", params)
        return response
