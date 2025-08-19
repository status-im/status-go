from uuid import uuid4
import pytest
from steps.messenger import MessengerSteps
from resources.enums import MessageContentType


@pytest.mark.rpc
class TestCreatePrivateGroups(MessengerSteps):

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        """Initialize two unprivileged backends (sender and receiver) for each test function"""
        self.sender = backend_new_profile("sender")
        self.receiver = backend_new_profile("receiver")
        self.make_contacts(self.sender, self.receiver)

    def test_create_group_chat_with_members(self):
        private_group_name = f"private_group_{uuid4()}"
        create_group_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], private_group_name)
        # TODO: Add more assertions on response

        self.get_message_by_content_type(
            create_group_response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=f"@{self.sender.public_key} created the group {private_group_name}",
        )[0]
        self.get_message_by_content_type(
            create_group_response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=f"@{self.sender.public_key} has added @{self.receiver.public_key}",
        )[0]

    def test_leave_group_chat(self):
        create_group_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], f"private_group_{uuid4()}")

        group_id = create_group_response.get("result", {}).get("chats", [])[0].get("id")
        members_before_leave = create_group_response.get("result", {}).get("chats", [])[0].get("members", [])
        assert len(members_before_leave) == 2
        assert self.sender.public_key in str(members_before_leave)

        leave_group_response = self.sender.wakuext_service.leave_group_chat(group_id, True)
        # TODO: Add more assertions on response

        self.get_message_by_content_type(
            leave_group_response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=f"@{self.sender.public_key} left the group",
        )[0]

        members_after_leave = leave_group_response.get("result", {}).get("chats", [])[0].get("members", [])
        assert len(members_after_leave) == 1
        assert self.sender.public_key not in str(members_after_leave)

    def test_send_group_chat_invitation_request(self, backend_new_profile):
        third_node = backend_new_profile("third_node")
        self.make_contacts(self.sender, third_node)

        create_group_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], f"private_group_{uuid4()}")
        group_id = create_group_response.get("result", {}).get("chats", [])[0].get("id")

        invitation_message = f"Please join {uuid4()}"
        invite_response = self.sender.wakuext_service.send_group_chat_invitation_request(group_id, third_node.public_key, invitation_message)
        # TODO: Add more assertions on response

        invitations = invite_response.get("result", {}).get("invitations", [])
        assert len(invitations) == 1
        assert invitations[0].get("chatId", "") == group_id
        assert invitations[0].get("from", "") == self.sender.public_key
        assert invitations[0].get("introductionMessage", "") == invitation_message
        assert invitations[0].get("state", 0) == 1

    def test_create_group_chat_from_invitation(self):
        invitation_group = f"Group name {uuid4()}"
        group_id = str(uuid4())
        create_from_inv = self.receiver.wakuext_service.create_group_chat_from_invitation(invitation_group, group_id, self.sender.public_key)
        # TODO: Add more assertions on response

        chats = create_from_inv.get("result", {}).get("chats", [])
        assert len(chats) == 1
        assert chats[0].get("id", "") == group_id
        assert chats[0].get("invitationAdmin", "") == self.sender.public_key
        assert chats[0].get("name", "") == invitation_group

    def test_add_members_to_group_chat(self, backend_new_profile):
        third_node = backend_new_profile("third_node")
        self.make_contacts(self.sender, third_node)

        create_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], f"add_members_group_{uuid4()}")
        group_id = create_response.get("result", {}).get("chats", [])[0].get("id")

        add_members_response = self.sender.wakuext_service.add_members_to_group_chat(group_id, [third_node.public_key])
        # TODO: Add more assertions on response

        self.get_message_by_content_type(
            add_members_response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=f"@{self.sender.public_key} has added @{third_node.public_key}",
        )[0]

    def test_remove_member_from_group_chat(self):
        create_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], f"group_{uuid4()}")
        group_id = create_response.get("result", {}).get("chats", [])[0].get("id")

        remove_member_response = self.sender.wakuext_service.remove_member_from_group_chat(group_id, self.receiver.public_key)
        # TODO: Add more assertions on response

        self.get_message_by_content_type(
            remove_member_response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=f"@{self.receiver.public_key} left the group",
        )[0]

    def test_remove_members_from_group_chat(self, backend_new_profile):
        third_node = backend_new_profile("third_node")
        self.make_contacts(self.sender, third_node)

        create_response = self.sender.wakuext_service.create_group_chat_with_members(
            [self.receiver.public_key, third_node.public_key], f"add_members_group_{uuid4()}"
        )
        group_id = create_response.get("result", {}).get("chats", [])[0].get("id")

        remove_members_response = self.sender.wakuext_service.remove_members_from_group_chat(
            group_id, [self.receiver.public_key, third_node.public_key]
        )
        # TODO: Add more assertions on response

        self.get_message_by_content_type(
            remove_members_response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=f"@{self.receiver.public_key} left the group",
        )[0]
        self.get_message_by_content_type(
            remove_members_response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=f"@{third_node.public_key} left the group",
        )[0]

    def test_confirm_joining_group(self):
        create_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], f"confirm_join_group_{uuid4()}")
        group_id = create_response.get("result", {}).get("chats", [])[0].get("id")

        confirm_response = self.sender.wakuext_service.confirm_joining_group(group_id)
        # TODO: Add more assertions on response

        chats = confirm_response.get("result", {}).get("chats", [])
        assert len(chats) == 1
        assert len(chats[0].get("members", [])) == 2

    def test_change_group_chat_name(self):
        initial_group_name = "initial_group_name"
        create_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], initial_group_name)
        group_id = create_response.get("result", {}).get("chats", [])[0].get("id")

        new_group_name = f"new_group_name_{uuid4()}"
        change_name_response = self.sender.wakuext_service.change_group_chat_name(group_id, new_group_name)
        # TODO: Add more assertions on response

        self.get_message_by_content_type(
            change_name_response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=f"@{self.sender.public_key} changed the group's name to {new_group_name}",
        )[0]

        chats = change_name_response.get("result", {}).get("chats", [])
        assert chats[0].get("name", "") == new_group_name

    @pytest.mark.skip(reason="waiting for https://github.com/status-im/status-go/issues/6752 resolution")
    def test_get_group_chat_invitations(self, backend_new_profile):
        third_node = backend_new_profile("third_node")
        self.make_contacts(self.sender, third_node)

        create_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], f"group_{uuid4()}")
        group_id = create_response.get("result", {}).get("chats", [])[0].get("id")

        self.sender.wakuext_service.send_group_chat_invitation_request(group_id, third_node.public_key, f"Please join {uuid4()}")
        third_node.wakuext_service.get_group_chat_invitations()
        # TODO: Add more assertions on response

    def test_send_group_chat_invitation_rejection(self, backend_new_profile):
        third_node = backend_new_profile("third_node")
        self.make_contacts(self.sender, third_node)

        create_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], f"group_{uuid4()}")
        group_id = create_response.get("result", {}).get("chats", [])[0].get("id")

        invitation_message = f"Please join {uuid4()}"
        send_invitation_response = self.sender.wakuext_service.send_group_chat_invitation_request(group_id, third_node.public_key, invitation_message)

        invitation_id = send_invitation_response.get("result", {}).get("invitations", [])[0].get("id")
        reject_response = self.sender.wakuext_service.send_group_chat_invitation_rejection(invitation_id)
        # TODO: Add more assertions on response

        invitations = reject_response.get("result", {}).get("invitations", [])
        assert len(invitations) == 1
        assert invitations[0].get("chatId", "") == group_id
        assert invitations[0].get("from", "") == self.sender.public_key
        assert invitations[0].get("introductionMessage", "") == invitation_message
        assert invitations[0].get("state", 0) == 2
