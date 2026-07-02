import pytest

from steps import messenger
from utils.rpc_helpers import messages_list
from waku_params import parametrize_waku_light_client


@pytest.mark.rpc
@parametrize_waku_light_client
class TestFetchingChatMessages:

    @pytest.fixture()
    def sender(self, backend_new_profile, waku_light_client):
        return backend_new_profile("sender", waku_light_client=waku_light_client)

    @pytest.fixture()
    def receiver(self, backend_new_profile, waku_light_client):
        return backend_new_profile("receiver", waku_light_client=waku_light_client)

    def test_chat_messages(self, sender, receiver):
        sent_texts, _ = messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)

        sender_chat_id = receiver.public_key
        response = sender.wakuext_service.chat_messages(sender_chat_id)
        # TODO: Add more assertions on response

        messages = response.get("messages", [])
        assert len(messages) == 1
        actual_text = messages[0].get("text", "")
        assert actual_text == sent_texts[0]

    def test_chat_messages_with_pagination(self, sender, receiver):
        sent_texts, _ = messenger.send_multiple_one_to_one_messages(5, sender=sender, receiver=receiver)
        sender_chat_id = receiver.public_key

        # Page 1
        chat_messages_res1 = sender.wakuext_service.chat_messages(sender_chat_id, cursor="", limit=3)

        cursor1 = chat_messages_res1.get("cursor", "")
        messages_page1 = chat_messages_res1.get("messages", [])
        assert len(messages_page1) == 3
        assert messages_page1[0].get("text", "") == sent_texts[4]
        assert messages_page1[1].get("text", "") == sent_texts[3]
        assert messages_page1[2].get("text", "") == sent_texts[2]
        assert cursor1 != ""

        # Page 2
        chat_messages_res2 = sender.wakuext_service.chat_messages(sender_chat_id, cursor=cursor1, limit=3)

        cursor2 = chat_messages_res2.get("cursor", "")
        messages_page2 = chat_messages_res2.get("messages", [])
        assert len(messages_page2) == 2
        assert messages_page2[0].get("text", "") == sent_texts[1]
        assert messages_page2[1].get("text", "") == sent_texts[0]
        assert cursor2 == ""

    def test_message_by_message_id(self, sender, receiver):
        sent_texts, responses = messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)

        message_id = responses[0].get("messages", [])[0].get("id", "")
        response = sender.wakuext_service.message_by_message_id(message_id)
        # TODO: Add more assertions on response

        actual_text = response.get("text", "")
        assert actual_text == sent_texts[0]

    @pytest.mark.parametrize(
        "search_term,case_sensitive,expected_count",
        [
            ("test_message_1", False, 1),
            ("TEST_MESSAGE_", False, 3),
            # ("TEST_MESSAGE_", True, 0), # Skipped due to https://github.com/status-im/status-go/issues/6359
        ],
    )
    def test_all_messages_from_chat_which_match_term(self, sender, receiver, search_term, case_sensitive, expected_count):
        messenger.send_multiple_one_to_one_messages(3, sender=sender, receiver=receiver)
        sender_chat_id = receiver.public_key

        response = sender.wakuext_service.all_messages_from_chat_which_match_term(sender_chat_id, search_term, case_sensitive)
        # TODO: Add more assertions on response

        messages = response.get("messages", [])
        assert len(messages) == expected_count

    def test_all_messages_from_chats_and_communities_which_match_term(self, sender, receiver):
        # One to one
        messenger.make_contacts(sender, receiver)
        sent_texts_one_to_one, _ = messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        one_to_one_chat_id = receiver.public_key

        # Group
        private_group_chat_id = messenger.join_private_group(admin=sender, member=receiver)
        text_group = "test_message_group"
        response = sender.wakuext_service.send_group_chat_message(private_group_chat_id, text_group)

        # Community
        community_id = messenger.create_community(sender)
        community_chat_id = messenger.join_community(member=receiver, admin=sender, community_id=community_id)
        text_community = "test_message_community"
        response = sender.wakuext_service.send_chat_message(community_chat_id, text_community)

        response = sender.wakuext_service.all_messages_from_chats_and_communities_which_match_term(
            [community_id], [one_to_one_chat_id, private_group_chat_id], "TEST_MESSAGE", False
        )
        # TODO: Add more assertions on response

        messages = messages_list(response)
        actual_texts = [message.get("text", "") for message in messages]
        assert sent_texts_one_to_one[0] in actual_texts
        assert text_group in actual_texts
        assert text_community in actual_texts

    @pytest.mark.skip(reason="Skipped due to https://github.com/status-im/status-go/issues/6359")
    def test_all_messages_from_chats_and_communities_which_match_term_case_sensitive(self, sender, receiver):
        # One to one
        messenger.make_contacts(sender, receiver)
        _, _ = messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        one_to_one_chat_id = receiver.public_key

        # Group
        private_group_chat_id = messenger.join_private_group(admin=sender, member=receiver)
        sender.wakuext_service.send_group_chat_message(private_group_chat_id, "test_message_group")

        # Community
        community_id = messenger.create_community(sender)
        community_chat_id = messenger.join_community(member=receiver, admin=sender, community_id=community_id)
        sender.wakuext_service.send_chat_message(community_chat_id, "test_message_community")

        response = sender.wakuext_service.all_messages_from_chats_and_communities_which_match_term(
            [community_id], [one_to_one_chat_id, private_group_chat_id], "TEST_MESSAGE", True
        )

        messages = response.get("messages", [])
        assert len(messages) == 0

    def test_first_unseen_message(self, sender, receiver):
        _, responses = messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        sender_chat_id = receiver.public_key
        message_id = responses[0].get("messages", [])[0].get("id", "")

        sender.wakuext_service.mark_message_as_unread(sender_chat_id, message_id)
        # TODO: Add more assertions on response

        result = sender.wakuext_service.first_unseen_message_id(sender_chat_id)
        # TODO: Add more assertions on response

        assert result == message_id
