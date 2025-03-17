import pytest

from steps.messenger import MessengerSteps

from clients.services.wakuext import SendPinMessagePayload


@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestInteractingWithChatMessages(MessengerSteps):

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
        self.sender.verify_json_schema(response, method="wakuext_chatPinnedMessages")

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

    def test_update_message_outgoing_status(self):
        _, responses = self.send_multiple_one_to_one_messages(1)
        message_id = responses[0].get("result", {}).get("messages", [])[0].get("id", "")
        new_status = "delivered"

        response = self.sender.wakuext_service.update_message_outgoing_status(message_id, new_status)
        self.sender.verify_json_schema(response, method="wakuext_updateMessageOutgoingStatus")

        response = self.sender.wakuext_service.message_by_message_id(message_id)
        outgoing_status = response.get("result", {}).get("outgoingStatus", "")
        assert outgoing_status == new_status
