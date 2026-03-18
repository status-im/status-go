from datetime import datetime, timezone, timedelta
from uuid import uuid4

import pytest

from clients.services.wakuext import ActivityCenterNotificationType
from clients.signals import SignalType
from resources.enums import MuteType
from steps.messenger import MessengerSteps


@pytest.mark.rpc
class TestCommunityChats(MessengerSteps):
    @pytest.fixture()
    def creator(self, backend_new_profile):
        return backend_new_profile("creator")

    @pytest.fixture()
    def member(self, backend_new_profile):
        return backend_new_profile("member")

    @pytest.fixture()
    def community_id(self, creator, member):
        cid = self.create_community(creator)
        self.join_community(member=member, admin=creator)
        return cid

    @pytest.fixture()
    def chat_payload(self):
        display_name = "chat_" + str(uuid4())
        return {
            "identity": {
                "displayName": display_name,
                "emoji": "😀",
                "color": "#1f2c75",
                "description": display_name,
            },
            "viewersCanPostReactions": False,
            "hideIfPermissionsNotMet": False,
            "permissions": {"access": 1},
        }

    def test_create_edit_delete_community_chat(self, creator, member, community_id, chat_payload):
        create_resp = creator.wakuext_service.create_community_chat(community_id, chat_payload)
        chats = create_resp.get("chats")
        communities = create_resp.get("communities")
        assert len(chats) == 1
        assert len(communities) == 1
        community_chats = communities[0].get("chats")

        # assert chats[0].get("displayName") == chat_payload["identity"]["displayName"] #  https://github.com/status-im/status-go/issues/6985
        assert chats[0].get("color") == chat_payload["identity"]["color"]
        assert chats[0].get("emoji") == chat_payload["identity"]["emoji"]
        assert chats[0].get("description") == chat_payload["identity"]["description"]
        assert chats[0].get("viewersCanPostReactions") == chat_payload["viewersCanPostReactions"]

        # build a dict keyed by position
        chats_by_position = {chat["position"]: chat for chat in community_chats.values()}
        # position 1 should be the default chat
        assert chats_by_position[0].get("name") == "general"
        # assert chats_by_position[1].get("name") == chat_payload["identity"]["displayName"] # https://github.com/status-im/status-go/issues/6985
        assert chats_by_position[1].get("hideIfPermissionsNotMet") == chat_payload["hideIfPermissionsNotMet"]
        assert chats_by_position[1].get("permissions") == chat_payload["permissions"]
        added_chat_id = chats_by_position[1].get("id")

        comm_before_delete = self.fetch_community(creator, community_id)
        assert added_chat_id in comm_before_delete.get("chats")

        edit_payload = {
            "identity": {
                "displayName": "edited_" + str(uuid4()),
                "emoji": "🔑",
                "color": "#212331",
                "description": "edited_" + str(uuid4()),
            }
        }
        edit_resp = creator.wakuext_service.edit_community_chat(community_id, added_chat_id, edit_payload)
        assert edit_resp.get("chats")[0].get("color") == edit_payload["identity"]["color"]
        assert edit_resp.get("chats")[0].get("emoji") == edit_payload["identity"]["emoji"]
        assert edit_resp.get("chats")[0].get("description") == edit_payload["identity"]["description"]

        del_resp = creator.wakuext_service.delete_community_chat(community_id, added_chat_id)
        assert del_resp.get("removedChats")[0] == added_chat_id
        assert added_chat_id not in del_resp.get("communities")[0].get("chats")

        comm_after_delete = self.fetch_community(creator, community_id)
        assert added_chat_id not in comm_after_delete.get("chats")

    def test_reorder_community_chat(self, creator, community_id, chat_payload):
        resp = creator.wakuext_service.create_community_chat(community_id, chat_payload)

        # sort chats by position
        chats_before = sorted(resp.get("communities")[0].get("chats").values(), key=lambda c: c["position"])

        first_chat = next(c for c in chats_before if c["position"] == 0)
        second_chat = next(c for c in chats_before if c["position"] == 1)

        # reorder: move the new chat to position 0
        reorder_resp = creator.wakuext_service.reorder_community_chat(community_id, second_chat["id"], 0)

        # fetch reordered chats and sort them again by position
        chats_after = sorted(reorder_resp["communities"][0]["chats"].values(), key=lambda c: c["position"])

        # assertions: positions must be swapped
        assert chats_after[0]["id"] == second_chat["id"]
        assert chats_after[0]["position"] == 0
        assert chats_after[1]["id"] == first_chat["id"]
        assert chats_after[1]["position"] == 1

    def test_mute_and_unmute_community_chats(self, creator, community_id, chat_payload):
        creator.wakuext_service.create_community_chat(community_id, chat_payload)

        community_before = creator.wakuext_service.fetch_community(community_id)
        assert community_before.get("muted") is False

        mute_resp = creator.wakuext_service.mute_community_chats(community_id, MuteType.MUTE_FOR15_MIN.value)
        assert mute_resp is not None

        community_after_mute = creator.wakuext_service.fetch_community(community_id)
        assert community_after_mute.get("muted") is True
        assert community_after_mute.get("muteTill") == mute_resp

        unmute_resp = creator.wakuext_service.un_mute_community_chats(community_id)
        assert unmute_resp is not None

        community_after_unmute = creator.wakuext_service.fetch_community(community_id)
        assert community_after_unmute.get("muted") is False
        assert community_after_unmute.get("muteTill") == "0001-01-01T00:00:00Z"

    @pytest.mark.parametrize("muted_type", [mt.value for mt in MuteType])
    def test_mute_types_are_applied(self, creator, member, community_id, chat_payload, muted_type):
        creator.wakuext_service.create_community_chat(community_id, chat_payload)

        mute_resp = member.wakuext_service.mute_community_chats(community_id, muted_type)
        community_after_mute = member.wakuext_service.fetch_community(community_id)
        muted_flag = community_after_mute.get("muted")
        comm_mute_till = community_after_mute.get("muteTill")

        if muted_type == MuteType.UNMUTED.value:
            assert muted_flag is False
        else:
            assert muted_flag is True

        assert comm_mute_till is not None
        assert str(comm_mute_till) == str(mute_resp)

        parsed_mute_till = datetime.fromisoformat(comm_mute_till.replace("Z", "+00:00"))

        now = datetime.now(timezone.utc)
        # time remaining until mute expires
        remaining = parsed_mute_till - now

        # map expected durations
        expected_map = {
            MuteType.MUTE_FOR15_MIN.value: timedelta(minutes=15),
            MuteType.MUTE_FOR1_HR.value: timedelta(hours=1),
            MuteType.MUTE_FOR8_HR.value: timedelta(hours=8),
            MuteType.MUTE_FOR24_HR.value: timedelta(hours=24),
            MuteType.MUTE_FOR1_WEEK.value: timedelta(days=7),
            MuteType.MUTE_TILL1_MIN.value: timedelta(minutes=1),
        }

        # allow some tolerance for clocks and propagation delays
        tolerance = timedelta(seconds=30)

        if muted_type in expected_map:
            expected = expected_map[muted_type]
            # remaining should be roughly equal to expected (allow +/- tolerance)
            assert abs(remaining - expected) <= tolerance, f"mute remaining {remaining}, expected ~{expected}"
        else:
            assert comm_mute_till == "0001-01-01T00:00:00Z"

    def test_send_community_chat_message_with_mention(self, creator, member, community_id, chat_payload):
        create_resp = creator.wakuext_service.create_community_chat(community_id, chat_payload)
        chat_id = create_resp.get("chats")[0].get("id")
        self.wait_for_community_chat_visible(member, community_id, chat_id)

        text = f"Hi @{member.public_key}"
        # creator sends a chat message with a mention to trigger a notification
        msg_start = len(member.received_signals[SignalType.MESSAGES_NEW])
        send_resp = creator.wakuext_service.send_chat_message(chat_id, text)
        assert send_resp.get("chats")[0].get("lastMessage").get("text") == text
        message_id = send_resp.get("messages", [])[0].get("id", "")

        # member receives that message even if chat is muted
        with member.expect_signal(SignalType.MESSAGES_NEW, pattern=message_id, timeout=10, start=msg_start):
            pass
        member_msgs_resp = member.wakuext_service.chat_messages(chat_id)
        assert member_msgs_resp.get("messages")[0].get("text") == text
        assert member_msgs_resp.get("messages")[0].get("mentioned") is True

        # member will receive the mention notification
        resp = member.wakuext_service.get_activity_center_notifications([activity for activity in ActivityCenterNotificationType])
        notifications = resp.get("notifications")
        assert len(notifications) == 2
        assert notifications[0].get("communityId") == community_id
        assert notifications[0].get("chatId") == chat_id
        assert notifications[0].get("message").get("text") == text

    def test_send_community_chat_message_while_chat_is_muted_and_then_unmuted(self, creator, member, community_id, chat_payload):
        create_resp = creator.wakuext_service.create_community_chat(community_id, chat_payload)
        chat_id = create_resp.get("chats")[0].get("id")
        self.wait_for_community_chat_visible(member, community_id, chat_id)

        # muting the community chats
        member.wakuext_service.mute_community_chats(community_id, MuteType.MUTE_FOR15_MIN.value)

        text = f"Hi @{member.public_key}"
        # creator sends a chat message with a mention to trigger a notification
        msg_start = len(member.received_signals[SignalType.MESSAGES_NEW])
        send_resp = creator.wakuext_service.send_chat_message(chat_id, text)
        assert send_resp.get("chats")[0].get("lastMessage").get("text") == text
        message_id = send_resp.get("messages", [])[0].get("id", "")

        # member receives that message even if chat is muted
        with member.expect_signal(SignalType.MESSAGES_NEW, pattern=message_id, timeout=10, start=msg_start):
            pass
        member_msgs_resp = member.wakuext_service.chat_messages(chat_id)
        assert member_msgs_resp.get("messages")[0].get("text") == text
        assert member_msgs_resp.get("messages")[0].get("mentioned") is True

        # member doesn't received the mention notification because the chat was muted
        resp = member.wakuext_service.get_activity_center_notifications([activity for activity in ActivityCenterNotificationType])
        notifications = resp.get("notifications")
        assert len(notifications) == 1
        assert notifications[0].get("chatId") == ""

        member.wakuext_service.un_mute_community_chats(community_id)

        msg_start = len(member.received_signals[SignalType.MESSAGES_NEW])
        send_resp = creator.wakuext_service.send_chat_message(chat_id, text)
        assert send_resp.get("chats")[0].get("lastMessage").get("text") == text
        message_id = send_resp.get("messages", [])[0].get("id", "")

        # member receives that message even if chat is muted
        with member.expect_signal(SignalType.MESSAGES_NEW, pattern=message_id, timeout=10, start=msg_start):
            pass
        member_msgs_resp = member.wakuext_service.chat_messages(chat_id)
        assert member_msgs_resp.get("messages")[0].get("text") == text
        assert member_msgs_resp.get("messages")[0].get("mentioned") is True

        # member receives again the mention notification because the chat was unmuted
        resp = member.wakuext_service.get_activity_center_notifications([activity for activity in ActivityCenterNotificationType])
        notifications = resp.get("notifications")
        assert len(notifications) == 2
        assert notifications[0].get("communityId") == community_id
        assert notifications[0].get("chatId") == chat_id
        assert notifications[0].get("message").get("text") == text
