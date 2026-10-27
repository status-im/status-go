from enum import Enum
from uuid import uuid4

import pytest

from clients.api import ApiResponseError
from clients.services.wakuext import ActivityCenterNotificationType
from clients.signals import SignalType
from steps import async_messenger
from waku_params import parametrize_waku_light_client, parametrize_waku_light_client_non_lc

THREAD_NAME_LIMIT = 50


class ChatKind(str, Enum):
    ONE_TO_ONE = "one_to_one"
    PRIVATE_GROUP = "private_group"
    COMMUNITY = "community"


async def _open_shared_chat(sender, receiver, chat_kind: ChatKind) -> tuple[str, str, str]:
    if chat_kind == ChatKind.ONE_TO_ONE:
        return receiver.public_key, sender.public_key, ""

    if chat_kind == ChatKind.PRIVATE_GROUP:
        await async_messenger.make_contacts(sender, receiver)
        chat_id = await async_messenger.join_private_group(admin=sender, member=receiver)
        return chat_id, chat_id, ""

    if chat_kind == ChatKind.COMMUNITY:
        community_id = async_messenger.create_community(sender)
        chat_id = await async_messenger.join_community(member=receiver, admin=sender, community_id=community_id)
        return chat_id, chat_id, community_id

    raise AssertionError(f"unsupported chat kind {chat_kind}")


def _message_by_text(response: dict, text: str) -> dict:
    for message in response.get("messages") or []:
        if message.get("text") == text:
            return message
    raise AssertionError(f"message {text!r} not found in {response.get('messages')}")


def _texts(response: dict) -> list[str]:
    return [message.get("text", "") for message in response.get("messages") or []]


def _thread(response: dict, thread_id: str) -> dict:
    threads = (response or {}).get("threads") or []
    for thread in threads:
        if thread.get("threadId") == thread_id:
            return thread
    raise AssertionError(f"thread {thread_id} not found in {threads}")


def _notification_by_message_id(response: dict, message_id: str) -> dict:
    for notification in (response or {}).get("notifications") or []:
        if (notification.get("message") or {}).get("id") == message_id:
            return notification
    raise AssertionError(f"notification for message {message_id} not found in {response}")


@pytest.mark.rpc
@pytest.mark.asyncio
@parametrize_waku_light_client
class TestWakuextThreads:

    @pytest.fixture
    async def sender(self, async_backend_new_profile, waku_light_client):
        return await async_backend_new_profile("sender", waku_light_client=waku_light_client)

    @pytest.fixture
    async def receiver(self, async_backend_new_profile, waku_light_client):
        return await async_backend_new_profile("receiver", waku_light_client=waku_light_client)

    @pytest.mark.parametrize("chat_kind", list(ChatKind), ids=lambda kind: kind.value)
    async def test_thread_reply_does_not_unread_parent_chat(self, sender, receiver, chat_kind):
        sender_chat_id, receiver_chat_id, community_id = await _open_shared_chat(sender, receiver, chat_kind)
        parent_text = f"parent-{uuid4()}-" + ("x" * THREAD_NAME_LIMIT)
        reply_text = f"reply-{uuid4()}"
        edited_reply_text = f"edited-{uuid4()}"

        if chat_kind == ChatKind.ONE_TO_ONE:
            parent_response = sender.wakuext_service.send_one_to_one_message(receiver.public_key, parent_text)
        else:
            parent_response = sender.wakuext_service.send_chat_message(sender_chat_id, parent_text)
        parent_id = _message_by_text(parent_response, parent_text)["id"]

        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=parent_id, timeout=60, check_buffer=True)
        receiver.wakuext_service.mark_all_read(receiver_chat_id)
        assert receiver.wakuext_service.chat(receiver_chat_id).get("unviewedMessagesCount") == 0

        created = sender.wakuext_service.create_thread(sender_chat_id, parent_id)
        thread = _thread(created, parent_id)
        assert thread["name"] == parent_text[:THREAD_NAME_LIMIT]
        assert thread["parentMessageId"] == parent_id

        reply_response = sender.wakuext_service.send_chat_message(sender_chat_id, reply_text, thread_id=parent_id)
        reply = _message_by_text(reply_response, reply_text)
        reply_id = reply["id"]
        assert reply.get("threadId") == parent_id
        assert not reply.get("responseTo")

        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=reply_id, timeout=60, check_buffer=True)

        parent_chat = receiver.wakuext_service.chat(receiver_chat_id)
        assert parent_chat.get("unviewedMessagesCount") == 0
        received_thread = _thread(receiver.wakuext_service.chat_threads(receiver_chat_id), parent_id)
        assert received_thread["unviewedMessagesCount"] == 1
        assert received_thread["name"] == parent_text[:THREAD_NAME_LIMIT]

        assert reply_text not in _texts(receiver.wakuext_service.chat_messages(receiver_chat_id, limit=20))
        thread_messages = receiver.wakuext_service.chat_messages_in_thread(receiver_chat_id, parent_id, limit=20)
        assert reply_text in _texts(thread_messages)

        summary = _thread(
            receiver.wakuext_service.chat_thread_summaries_by_parent_message_ids(receiver_chat_id, [parent_id]),
            parent_id,
        )
        assert summary["messagesCount"] == 2
        assert summary["lastMessage"]["text"] == reply_text
        assert summary["participantsCount"] >= 1
        assert summary["participantsPreviewIds"]

        with pytest.raises(ApiResponseError, match="can't pin a message in a thread"):
            sender.wakuext_service.send_pin_message({"chat_id": sender_chat_id, "message_id": reply_id, "pinned": True})

        if chat_kind == ChatKind.COMMUNITY:
            marked = receiver.wakuext_service.mark_all_read_in_community(community_id)
        else:
            marked = receiver.wakuext_service.mark_thread_read(receiver_chat_id, parent_id)
        assert _thread(marked, parent_id)["unviewedMessagesCount"] == 0
        assert receiver.wakuext_service.chat(receiver_chat_id).get("unviewedMessagesCount") == 0
        assert _thread(receiver.wakuext_service.chat_threads(receiver_chat_id), parent_id)["unviewedMessagesCount"] == 0

        sender.wakuext_service.edit_message(reply_id, edited_reply_text)
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=edited_reply_text, timeout=60, check_buffer=True)
        assert receiver.wakuext_service.message_by_message_id(reply_id).get("text") == edited_reply_text
        assert receiver.wakuext_service.message_by_message_id(reply_id).get("threadId") == parent_id

        with pytest.raises(ApiResponseError, match="already in a thread"):
            sender.wakuext_service.create_thread(sender_chat_id, reply_id)


async def _one_to_one_thread(sender, receiver, parent_text: str, reply_text: str) -> tuple[str, str, str, str]:
    sender_chat_id, receiver_chat_id, _ = await _open_shared_chat(sender, receiver, ChatKind.ONE_TO_ONE)
    parent_response = sender.wakuext_service.send_one_to_one_message(receiver.public_key, parent_text)
    parent_id = _message_by_text(parent_response, parent_text)["id"]
    await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=parent_id, timeout=60, check_buffer=True)
    sender.wakuext_service.create_thread(sender_chat_id, parent_id)
    reply_response = sender.wakuext_service.send_chat_message(sender_chat_id, reply_text, thread_id=parent_id)
    reply_id = _message_by_text(reply_response, reply_text)["id"]
    await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=reply_id, timeout=60, check_buffer=True)
    return sender_chat_id, receiver_chat_id, parent_id, reply_id


@pytest.mark.rpc
@pytest.mark.asyncio
@parametrize_waku_light_client_non_lc
class TestWakuextThreadExtras:

    @pytest.fixture
    async def sender(self, async_backend_new_profile, waku_light_client):
        return await async_backend_new_profile("sender", waku_light_client=waku_light_client)

    @pytest.fixture
    async def receiver(self, async_backend_new_profile, waku_light_client):
        return await async_backend_new_profile("receiver", waku_light_client=waku_light_client)

    async def test_start_thread_from_new_message(self, sender, receiver):
        sender_chat_id = receiver.public_key
        receiver_chat_id = sender.public_key
        seed_text = f"seed-{uuid4()}"
        sender.wakuext_service.send_one_to_one_message(receiver.public_key, seed_text)
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=seed_text, timeout=60, check_buffer=True)

        thread_name = f"Named-{uuid4()}"
        reply_text = f"first-{uuid4()}"
        started = sender.wakuext_service.start_thread_from_new_message(sender_chat_id, reply_text, thread_name)
        root = _message_by_text(started, thread_name)
        reply = _message_by_text(started, reply_text)
        thread = _thread(started, root["id"])
        assert thread["name"] == thread_name
        assert thread["parentMessageId"] == root["id"]
        assert not root.get("threadId")
        assert reply.get("threadId") == root["id"]

        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=reply["id"], timeout=60, check_buffer=True)
        assert thread_name in _texts(receiver.wakuext_service.chat_messages(receiver_chat_id, limit=20))
        assert reply_text not in _texts(receiver.wakuext_service.chat_messages(receiver_chat_id, limit=20))
        assert reply_text in _texts(receiver.wakuext_service.chat_messages_in_thread(receiver_chat_id, root["id"], limit=20))
        received = _thread(receiver.wakuext_service.chat_threads(receiver_chat_id), root["id"])
        assert received["name"] == thread_name

    async def test_thread_reply_reaction_stays_in_thread(self, sender, receiver):
        parent_text = f"parent-{uuid4()}"
        reply_text = f"reply-{uuid4()}"
        sender_chat_id, receiver_chat_id, parent_id, reply_id = await _one_to_one_thread(sender, receiver, parent_text, reply_text)

        reaction = receiver.wakuext_service.send_emoji_reaction(receiver_chat_id, reply_id, "2764")
        emoji_id = reaction["emojiReactions"][0]["id"]
        await sender.wait_for_signal(SignalType.MESSAGES_NEW, pattern=emoji_id, timeout=60, check_buffer=True)

        parent_reactions = sender.wakuext_service.emoji_reactions_by_chat_id(sender_chat_id, 20)
        assert all(item.get("messageId") != reply_id for item in parent_reactions)
        thread_reactions = sender.wakuext_service.emoji_reactions_by_chat_id_v2(sender_chat_id, parent_id, 20)
        assert any(item.get("messageId") == reply_id and item.get("emoji") == "2764" for item in thread_reactions)

    async def test_thread_search_finds_reply_not_sibling(self, sender, receiver):
        chat_id, _, _ = await _open_shared_chat(sender, receiver, ChatKind.PRIVATE_GROUP)
        parent_text = f"parent-{uuid4()}"
        reply_text = f"reply-{uuid4()}"
        sibling_text = f"sibling-{uuid4()}"
        parent_response = sender.wakuext_service.send_chat_message(chat_id, parent_text)
        parent_id = _message_by_text(parent_response, parent_text)["id"]
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=parent_id, timeout=60, check_buffer=True)
        sender.wakuext_service.create_thread(chat_id, parent_id)
        reply_response = sender.wakuext_service.send_chat_message(chat_id, reply_text, thread_id=parent_id)
        reply_id = _message_by_text(reply_response, reply_text)["id"]
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=reply_id, timeout=60, check_buffer=True)
        sender.wakuext_service.send_chat_message(chat_id, sibling_text)
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=sibling_text, timeout=60, check_buffer=True)

        found = receiver.wakuext_service.all_messages_from_thread_which_match_term(chat_id, parent_id, reply_text, False)
        assert reply_text in _texts(found)
        missed = receiver.wakuext_service.all_messages_from_thread_which_match_term(chat_id, parent_id, sibling_text, False)
        assert sibling_text not in _texts(missed)


@pytest.mark.rpc
@pytest.mark.asyncio
@parametrize_waku_light_client
class TestWakuextThreadRename:

    @pytest.fixture
    async def sender(self, async_backend_new_profile, waku_light_client):
        return await async_backend_new_profile("sender", waku_light_client=waku_light_client)

    @pytest.fixture
    async def receiver(self, async_backend_new_profile, waku_light_client):
        return await async_backend_new_profile("receiver", waku_light_client=waku_light_client)

    async def test_thread_rename_reaches_the_peer(self, sender, receiver):
        parent_text = f"parent-{uuid4()}"
        reply_text = f"reply-{uuid4()}"
        renamed = f"Renamed-{uuid4()}"
        sender_chat_id, receiver_chat_id, parent_id, _ = await _one_to_one_thread(sender, receiver, parent_text, reply_text)

        edited = sender.wakuext_service.edit_thread(sender_chat_id, parent_id, renamed)
        assert _thread(edited, parent_id)["name"] == renamed

        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=renamed, timeout=60, check_buffer=True)
        assert _thread(receiver.wakuext_service.chat_threads(receiver_chat_id), parent_id)["name"] == renamed
        assert receiver.wakuext_service.message_by_message_id(parent_id).get("text") == parent_text


@pytest.mark.rpc
@pytest.mark.asyncio
class TestWakuextThreadActivityCenter:

    @pytest.fixture
    async def sender(self, async_backend_new_profile):
        return await async_backend_new_profile("sender")

    @pytest.fixture
    async def receiver(self, async_backend_new_profile):
        return await async_backend_new_profile("receiver")

    @pytest.mark.parametrize("chat_kind", [ChatKind.PRIVATE_GROUP, ChatKind.COMMUNITY], ids=lambda kind: kind.value)
    async def test_mention_and_reply_in_thread_expose_thread_name(self, sender, receiver, chat_kind):
        sender_chat_id, receiver_chat_id, _ = await _open_shared_chat(sender, receiver, chat_kind)
        parent_text = f"Release-{uuid4()}"
        mention_text = f"Hello @{receiver.public_key}"
        reply_text = f"reply-{uuid4()}"
        renamed = f"Renamed-{uuid4()}"

        if chat_kind == ChatKind.COMMUNITY:
            open_chat_text = f"open-chat-{uuid4()}"
            sender.wakuext_service.send_chat_message(sender_chat_id, open_chat_text)
            await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=open_chat_text, timeout=60, check_buffer=True)

        parent_response = receiver.wakuext_service.send_chat_message(receiver_chat_id, parent_text)
        parent_id = _message_by_text(parent_response, parent_text)["id"]
        await sender.wait_for_signal(SignalType.MESSAGES_NEW, pattern=parent_id, timeout=60, check_buffer=True)

        sender.wakuext_service.create_thread(sender_chat_id, parent_id)
        expected_name = parent_text[:THREAD_NAME_LIMIT]

        mention_response = sender.wakuext_service.send_chat_message(sender_chat_id, mention_text, thread_id=parent_id)
        mention_id = _message_by_text(mention_response, mention_text)["id"]
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=mention_id, timeout=60, check_buffer=True)

        reply_response = sender.wakuext_service.send_chat_message(sender_chat_id, reply_text, thread_id=parent_id, responseTo=parent_id)
        reply_id = _message_by_text(reply_response, reply_text)["id"]
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=reply_id, timeout=60, check_buffer=True)

        mention = _notification_by_message_id(
            receiver.wakuext_service.get_activity_center_notifications(
                activity_types=[ActivityCenterNotificationType.NOTIFICATION_TYPE_MENTION],
            ),
            mention_id,
        )
        assert mention["type"] == ActivityCenterNotificationType.NOTIFICATION_TYPE_MENTION.value
        assert mention["message"]["threadId"] == parent_id
        assert mention["threadName"] == expected_name

        reply = _notification_by_message_id(
            receiver.wakuext_service.get_activity_center_notifications(
                activity_types=[ActivityCenterNotificationType.NOTIFICATION_TYPE_REPLY],
            ),
            reply_id,
        )
        assert reply["type"] == ActivityCenterNotificationType.NOTIFICATION_TYPE_REPLY.value
        assert reply["message"]["threadId"] == parent_id
        assert reply["threadName"] == expected_name

        sender.wakuext_service.edit_thread(sender_chat_id, parent_id, renamed)
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=renamed, timeout=60, check_buffer=True)
        renamed_mention = _notification_by_message_id(
            receiver.wakuext_service.get_activity_center_notifications(
                activity_types=[ActivityCenterNotificationType.NOTIFICATION_TYPE_MENTION],
            ),
            mention_id,
        )
        assert renamed_mention["threadName"] == renamed
