from enum import Enum


class MessageContentType(Enum):
    UNKNOWN_CONTENT_TYPE = 0
    TEXT_PLAIN = 1
    STICKER = 2
    STATUS = 3
    EMOJI = 4
    TRANSACTION_COMMAND = 5
    SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP = 6
    IMAGE = 7
    AUDIO = 8
    COMMUNITY = 9
    SYSTEM_MESSAGE_GAP = 10
    CONTACT_REQUEST = 11
    DISCORD_MESSAGE = 12
    IDENTITY_VERIFICATION = 13
    SYSTEM_MESSAGE_PINNED_MESSAGE = 14
    SYSTEM_MESSAGE_MUTUAL_EVENT_SENT = 15
    SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED = 16
    SYSTEM_MESSAGE_MUTUAL_EVENT_REMOVED = 17
    BRIDGE_MESSAGE = 18


class ChatType(Enum):
    UNKNOWN_TYPE = 0
    ONE_TO_ONE = 1
    PUBLIC = 2
    PRIVATE_GROUP_CHAT = 3
    PROFILE = 4  # Deprecated
    TIMELINE = 5  # Deprecated
    COMMUNITY_CHAT = 6


class MuteType(Enum):
    MUTE_FOR15_MIN = 1
    MUTE_FOR1_HR = 2
    MUTE_FOR8_HR = 3
    MUTE_FOR1_WEEK = 4
    MUTE_TILL_UNMUTED = 5
    MUTE_TILL1_MIN = 6
    UNMUTED = 7
    MUTE_FOR24_HR = 8


class ChatPreviewFilterType(Enum):
    Community = 0
    NonCommunity = 1


class RequestToJoinState(Enum):
    RequestToJoinStatePending = 1
    RequestToJoinStateDeclined = 2
    RequestToJoinStateAccepted = 3
    RequestToJoinStateCanceled = 4


class ContactVerificationState(Enum):
    Pending = 1
    Accepted = 2
    Declined = 3
    Canceled = 4


class StatusType(Enum):
    # protobuf.StatusUpdate StatusType values (see protocol/protobuf/status_update.proto).
    # Only AUTOMATIC status updates are published as ephemeral Waku messages, so they
    # are never persisted by the store node.
    UNKNOWN_STATUS_TYPE = 0
    AUTOMATIC = 1  # ephemeral
    DO_NOT_DISTURB = 2  # persisted
    ALWAYS_ONLINE = 3
    INACTIVE = 4
