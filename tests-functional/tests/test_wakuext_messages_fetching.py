import pytest

from steps.messenger import MessengerSteps


@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestFetchingChatMessages(MessengerSteps):
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
