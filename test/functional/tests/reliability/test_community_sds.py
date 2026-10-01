"""Community SDS outgoing-status regression."""

from uuid import uuid4

import pytest

from clients.signals import SignalType
from steps import messenger

SDS_TIMEOUT = 60
SDS_IMAGE_ALBUM_SIZE = 5


def _member_ack(member, chat_id: str):
    member.wakuext_service.send_chat_message(chat_id, f"sds_ack_{uuid4()}")


def _expect_sds_delivered(admin, message_id: str):
    with admin.expect_signal(SignalType.MESSAGE_DELIVERED, pattern=message_id, timeout=SDS_TIMEOUT, start="beginning"):
        pass
    messenger.assert_outgoing_status(admin, message_id, "delivered")


def _assert_stays_sent_not_delivered(admin, message_id: str, delivered_start: int):
    messenger.assert_outgoing_status(admin, message_id, "sent")
    messenger.assert_no_signal_with_pattern(admin, SignalType.MESSAGE_DELIVERED, message_id, delivered_start)


@pytest.mark.reliability
@pytest.mark.sds
class TestCommunitySDS:
    def test_community_sds_semantics(self, community_chat):
        admin = community_chat["admin"]
        member = community_chat["member"]
        chat_id = community_chat["chat_id"]

        delivered_start = len(admin.received_signals[SignalType.MESSAGE_DELIVERED])
        response = admin.wakuext_service.send_chat_message(chat_id, f"sds_no_ack_{uuid4()}")
        online_id = messenger.get_message_id(response)
        with member.expect_signal(SignalType.MESSAGES_NEW, pattern=online_id, timeout=SDS_TIMEOUT, start="beginning"):
            pass
        with admin.expect_signal(SignalType.ENVELOPE_SENT, pattern=online_id, timeout=SDS_TIMEOUT, start="beginning"):
            pass
        _assert_stays_sent_not_delivered(admin, online_id, delivered_start)

        delivered_start = len(admin.received_signals[SignalType.MESSAGE_DELIVERED])
        with messenger.node_pause(member):
            response = admin.wakuext_service.send_chat_message(chat_id, f"sds_offline_{uuid4()}")
            offline_id = messenger.get_message_id(response)
            with admin.expect_signal(SignalType.ENVELOPE_SENT, pattern=offline_id, timeout=SDS_TIMEOUT, start="beginning"):
                pass
            _assert_stays_sent_not_delivered(admin, offline_id, delivered_start)

        response = admin.wakuext_service.send_chat_message(chat_id, f"sds_delivered_{uuid4()}")
        text_id = messenger.get_message_id(response)
        with member.expect_signal(SignalType.MESSAGES_NEW, pattern=text_id, timeout=SDS_TIMEOUT, start="beginning"):
            pass
        _member_ack(member, chat_id)
        _expect_sds_delivered(admin, text_id)

        image_path = messenger.import_test_image(admin)
        messenger.import_test_image(member)
        sent_messages = messenger.community_image_messages(chat_id, image_path, message_count=SDS_IMAGE_ALBUM_SIZE, sender=admin, receiver=member)
        image_id = sent_messages[0].get("id")
        assert image_id
        _member_ack(member, chat_id)
        _expect_sds_delivered(admin, image_id)
