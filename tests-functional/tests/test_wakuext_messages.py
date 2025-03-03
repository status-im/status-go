import pytest

from tests.test_cases import MessengerTestCase

from clients.services.wakuext import SendPinMessagePayload
from clients.signals import SignalType


@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestChatMessages(MessengerTestCase):

    def test_chat_messages(self):
        sent_texts, _ = self.send_multiple_one_to_one_messages(1)

        sender_chat_id = self.receiver.public_key
        response = self.sender.wakuext_service.chat_messages(sender_chat_id)

        self.sender.verify_json_schema(response, method="wakuext_chatMessages")

        messages = response.get("result", {}).get("messages", [])
        assert len(messages) == 1
        actual_text = messages[0].get("text", "")
        assert actual_text == sent_texts[0]

    def test_chat_messages_with_pagination(self):
        sent_texts, _ = self.send_multiple_one_to_one_messages(5)
        sender_chat_id = self.receiver.public_key

        # Page 1
        chat_messages_res1 = self.sender.wakuext_service.chat_messages(sender_chat_id, cursor="", limit=3)

        cursor1 = chat_messages_res1.get("result", {}).get("cursor", "")
        messages_page1 = chat_messages_res1.get("result", {}).get("messages", [])
        assert len(messages_page1) == 3
        assert messages_page1[0].get("text", "") == sent_texts[4]
        assert messages_page1[1].get("text", "") == sent_texts[3]
        assert messages_page1[2].get("text", "") == sent_texts[2]
        assert cursor1 != ""

        # Page 2
        chat_messages_res2 = self.sender.wakuext_service.chat_messages(sender_chat_id, cursor=cursor1, limit=3)

        cursor2 = chat_messages_res2.get("result", {}).get("cursor", "")
        messages_page2 = chat_messages_res2.get("result", {}).get("messages", [])
        assert len(messages_page2) == 2
        assert messages_page2[0].get("text", "") == sent_texts[1]
        assert messages_page2[1].get("text", "") == sent_texts[0]
        assert cursor2 == ""

    def test_message_by_message_id(self):
        sent_texts, responses = self.send_multiple_one_to_one_messages(1)

        message_id = responses[0].get("result", {}).get("messages", [])[0].get("id", "")
        response = self.sender.wakuext_service.message_by_message_id(message_id)

        self.sender.verify_json_schema(response, method="wakuext_messageByMessageID")

        actual_text = response.get("result", {}).get("text", "")
        assert actual_text == sent_texts[0]

    @pytest.mark.parametrize(
        "searchTerm,caseSensitive,expectedCount",
        [
            ("test_message_1", False, 1),
            ("TEST_MESSAGE_", False, 3),
            # ("TEST_MESSAGE_", True, 0),  # Skipped due to https://github.com/status-im/status-go/issues/6359
        ],
    )
    def test_all_messages_from_chat_which_match_term(self, searchTerm, caseSensitive, expectedCount):
        self.send_multiple_one_to_one_messages(3)
        sender_chat_id = self.receiver.public_key

        response = self.sender.wakuext_service.all_messages_from_chat_which_match_term(sender_chat_id, searchTerm, caseSensitive)

        self.sender.verify_json_schema(response, method="wakuext_allMessagesFromChatWhichMatchTerm")

        messages = response.get("result", {}).get("messages", [])
        assert len(messages) == expectedCount

    def test_pinned_messages(self):
        sent_texts, responses = self.send_multiple_one_to_one_messages(1)

        # pin
        message = responses[0].get("result", {}).get("messages", [])[0]
        pin_message_payload: SendPinMessagePayload = {
            "chat_id": message.get("chatId", ""),
            "message_id": message.get("id", ""),
            "pinned": True,
        }

        response = self.sender.wakuext_service.send_pin_message(pin_message_payload)
        self.sender.verify_json_schema(response, method="wakuext_sendPinMessage")

        sender_chat_id = self.receiver.public_key
        response = self.sender.wakuext_service.chat_pinned_messages(sender_chat_id)
        self.sender.verify_json_schema(response, method="wakuext_schatPinnedMessages")

        pinned_messages = response.get("result", {}).get("pinnedMessages", [])
        assert len(pinned_messages) == 1
        actual_text = pinned_messages[0].get("message", {}).get("text", "")
        assert actual_text == sent_texts[0]

        # unpin
        pin_message_payload["pinned"] = False
        self.sender.wakuext_service.send_pin_message(pin_message_payload)
        response = self.sender.wakuext_service.chat_pinned_messages(sender_chat_id)

        pinned_messages = response.get("result", {}).get("pinnedMessages", [])
        assert pinned_messages is None

    def test_pinned_messages_with_pagination(self):
        sent_texts, responses = self.send_multiple_one_to_one_messages(5)
        sender_chat_id = self.receiver.public_key

        for response in responses:
            message = response.get("result", {}).get("messages", [])[0]
            pin_message_payload: SendPinMessagePayload = {
                "chat_id": message.get("chatId", ""),
                "message_id": message.get("id", ""),
                "pinned": True,
            }
            self.sender.wakuext_service.send_pin_message(pin_message_payload)

        # Page 1
        pinned_messages_res1 = self.sender.wakuext_service.chat_pinned_messages(sender_chat_id, cursor="", limit=3)

        cursor1 = pinned_messages_res1.get("result", {}).get("cursor", "")
        pinned_messages_page1 = pinned_messages_res1.get("result", {}).get("pinnedMessages", [])
        assert len(pinned_messages_page1) == 3
        assert pinned_messages_page1[0].get("message", {}).get("text", "") == sent_texts[4]
        assert pinned_messages_page1[1].get("message", {}).get("text", "") == sent_texts[3]
        assert pinned_messages_page1[2].get("message", {}).get("text", "") == sent_texts[2]
        assert cursor1 != ""

        # Page 2
        pinned_messages_res2 = self.sender.wakuext_service.chat_pinned_messages(sender_chat_id, cursor=cursor1, limit=3)

        cursor2 = pinned_messages_res2.get("result", {}).get("cursor", "")
        pinned_messages_page2 = pinned_messages_res2.get("result", {}).get("pinnedMessages", [])
        assert len(pinned_messages_page2) == 2
        assert pinned_messages_page2[0].get("message", {}).get("text", "") == sent_texts[1]
        assert pinned_messages_page2[1].get("message", {}).get("text", "") == sent_texts[0]
        assert cursor2 == ""

    def test_edit_message(self):
        sent_texts, responses = self.send_multiple_one_to_one_messages(1)
        message_id = responses[0].get("result", {}).get("messages", [])[0].get("id", "")

        response = self.sender.wakuext_service.message_by_message_id(message_id)
        actual_text = response.get("result", {}).get("text", "")
        assert actual_text == sent_texts[0]

        new_text = "test_message_edited"
        response = self.sender.wakuext_service.edit_message(message_id, new_text)
        self.sender.verify_json_schema(response, method="wakuext_editMessage")

        response = self.sender.wakuext_service.message_by_message_id(message_id)
        actual_text = response.get("result", {}).get("text", "")
        assert actual_text == new_text

    def test_delete_message(self):
        _, responses = self.send_multiple_one_to_one_messages(1)

        message_id = responses[0].get("result", {}).get("messages", [])[0].get("id", "")
        response = self.sender.wakuext_service.message_by_message_id(message_id)
        assert response.get("result", {}) != {}

        response = self.sender.wakuext_service.delete_message(message_id)
        self.sender.verify_json_schema(response, method="wakuext_deleteMessage")

        response = self.sender.wakuext_service.message_by_message_id(message_id, skip_validation=True)
        error_code = response.get("error", {}).get("code", 0)
        error_message = response.get("error", {}).get("message", "")
        assert error_code == -32000
        assert error_message == "record not found"

    def test_delete_message_and_send(self):
        _, responses = self.send_multiple_one_to_one_messages(1)

        message_id = responses[0].get("result", {}).get("messages", [])[0].get("id", "")
        response = self.sender.wakuext_service.message_by_message_id(message_id)
        assert response.get("result", {}) != {}

        response = self.sender.wakuext_service.delete_message_and_send(message_id)
        self.sender.verify_json_schema(response, method="wakuext_deleteMessageAndSend")
        removed_messages = response.get("result", {}).get("removedMessages", [])
        assert len(removed_messages) == 1
        assert removed_messages[0].get("messageId") == message_id

        response = self.sender.wakuext_service.message_by_message_id(message_id)
        message = response.get("result", {})
        assert message.get("id", "") == message_id
        assert message.get("deleted", None) is True

    def test_delete_messages_by_chat_id(self):
        _, _ = self.send_multiple_one_to_one_messages(3)
        sender_chat_id = self.receiver.public_key

        response = self.sender.wakuext_service.chat_messages(sender_chat_id)
        messages = response.get("result", {}).get("messages", [])
        assert len(messages) == 3

        response = self.sender.wakuext_service.delete_messages_by_chat_id(sender_chat_id)
        self.sender.verify_json_schema(response, method="wakuext_deleteMessagesByChatID")

        response = self.sender.wakuext_service.chat_messages(sender_chat_id)
        messages = response.get("result", {}).get("messages", [])
        assert messages is None

    def test_delete_message_for_me_and_sync(self):
        _, responses = self.send_multiple_one_to_one_messages(1)

        message_id = responses[0].get("result", {}).get("messages", [])[0].get("id", "")
        local_chat_id = responses[0].get("result", {}).get("messages", [])[0].get("localChatId", "")
        response = self.sender.wakuext_service.message_by_message_id(message_id)
        assert response.get("result", {}) != {}

        response = self.sender.wakuext_service.delete_message_for_me_and_sync(local_chat_id, message_id)
        self.sender.verify_json_schema(response, method="wakuext_deleteMessageForMeAndSync")

        response = self.sender.wakuext_service.message_by_message_id(message_id)
        message = response.get("result", {})
        assert message.get("id", "") == message_id
        assert message.get("deletedForMe", None) is True

        # TODO: assert sync action

    def test_first_unseen_message(self):
        _, responses = self.send_multiple_one_to_one_messages(1)
        sender_chat_id = self.receiver.public_key
        message_id = responses[0].get("result", {}).get("messages", [])[0].get("id", "")

        response = self.sender.wakuext_service.mark_message_as_unread(sender_chat_id, message_id)
        self.sender.verify_json_schema(response, method="wakuext_markMessageAsUnread")

        response = self.sender.wakuext_service.first_unseen_message_id(sender_chat_id)
        self.sender.verify_json_schema(response, method="wakuext_firstUnseenMessageID")

        result = response.get("result", "")
        assert result == message_id


@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestUserStatus(MessengerTestCase):

    def test_status_updates(self):
        self.make_contacts()

        statuses = [[1, "text_1"], [2, "text_2"], [3, "text_3"], [4, "text_4"]]

        for new_status, custom_text in statuses:
            response = self.sender.wakuext_service.set_user_status(new_status, custom_text)
            self.sender.verify_json_schema(response, method="wakuext_setUserStatus")

            self.receiver.find_signal_containing_pattern(
                SignalType.MESSAGES_NEW.value,
                event_pattern=custom_text,
                timeout=10,
            )

            response = self.receiver.wakuext_service.status_updates()
            self.sender.verify_json_schema(response, method="wakuext_statusUpdates")

            statusUpdate = response.get("result", {}).get("statusUpdates", [])[0]
            assert statusUpdate.get("statusType", -1) == new_status
            assert statusUpdate.get("text", "") == custom_text
