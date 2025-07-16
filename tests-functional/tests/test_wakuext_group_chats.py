from uuid import uuid4
import pytest
from steps.messenger import MessengerSteps
from resources.enums import MessageContentType


@pytest.mark.rpc
class TestCreatePrivateGroups(MessengerSteps):

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_factory):
        """Initialize two unprivileged backends (sender and receiver) for each test function"""
        self.sender = backend_factory("sender")
        self.receiver = backend_factory("receiver")
        self.make_contacts(self.sender, self.receiver)

    def test_create_group_chat_with_members(self):
        private_group_name = f"private_group_{uuid4()}"
        create_group_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], private_group_name)
        self.sender.verify_json_schema(create_group_response, method="wakuext_createGroupChatWithMembers")

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
        leave_group_response = self.sender.wakuext_service.leave_group_chat(group_id, True)
        self.sender.verify_json_schema(leave_group_response, method="wakuext_leaveGroupChat")

        self.get_message_by_content_type(
            leave_group_response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=f"@{self.sender.public_key} left the group",
        )[0]

    def test_send_group_chat_invitation_request(self, backend_factory):
        third_node = backend_factory("third_node")
        self.make_contacts(self.sender, third_node)

        create_group_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], f"private_group_{uuid4()}")
        group_id = create_group_response.get("result", {}).get("chats", [])[0].get("id")

        invitation_message = f"Please join {uuid4()}"
        invite_response = self.sender.wakuext_service.send_group_chat_invitation_request(group_id, third_node.public_key, invitation_message)
        self.sender.verify_json_schema(invite_response, method="wakuext_sendGroupChatInvitationRequest")

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
        self.sender.verify_json_schema(create_from_inv, method="wakuext_createGroupChatFromInvitation")

        chats = create_from_inv.get("result", {}).get("chats", [])
        assert len(chats) == 1
        assert chats[0].get("id", "") == group_id
        assert chats[0].get("invitationAdmin", "") == self.sender.public_key
        assert chats[0].get("name", "") == invitation_group

    def test_add_members_to_group_chat(self, backend_factory):
        third_node = backend_factory("third_node")
        self.make_contacts(self.sender, third_node)

        create_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], f"add_members_group_{uuid4()}")
        group_id = create_response.get("result", {}).get("chats", [])[0].get("id")

        add_members_response = self.sender.wakuext_service.add_members_to_group_chat(group_id, [third_node.public_key])
        self.sender.verify_json_schema(add_members_response, method="wakuext_addMembersToGroupChat")

        self.get_message_by_content_type(
            add_members_response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=f"@{self.sender.public_key} has added @{third_node.public_key}",
        )[0]

    def test_remove_member_from_group_chat(self):
        create_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], f"group_{uuid4()}")
        group_id = create_response.get("result", {}).get("chats", [])[0].get("id")

        remove_member_response = self.sender.wakuext_service.remove_member_from_group_chat(group_id, self.receiver.public_key)
        self.sender.verify_json_schema(remove_member_response, method="wakuext_removeMemberFromGroupChat")

        self.get_message_by_content_type(
            remove_member_response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=f"@{self.receiver.public_key} left the group",
        )[0]

    def test_remove_members_from_group_chat(self, backend_factory):
        third_node = backend_factory("third_node")
        self.make_contacts(self.sender, third_node)

        create_response = self.sender.wakuext_service.create_group_chat_with_members(
            [self.receiver.public_key, third_node.public_key], f"add_members_group_{uuid4()}"
        )
        group_id = create_response.get("result", {}).get("chats", [])[0].get("id")

        remove_members_response = self.sender.wakuext_service.remove_members_from_group_chat(
            group_id, [self.receiver.public_key, third_node.public_key]
        )
        self.sender.verify_json_schema(remove_members_response, method="wakuext_removeMembersFromGroupChat")

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
        self.sender.verify_json_schema(confirm_response, method="wakuext_confirmJoiningGroup")

        chats = confirm_response.get("result", {}).get("chats", [])
        assert len(chats) == 1
        assert len(chats[0].get("members", [])) == 2

    def test_change_group_chat_name(self):
        initial_group_name = "initial_group_name"
        create_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], initial_group_name)
        group_id = create_response.get("result", {}).get("chats", [])[0].get("id")

        new_group_name = f"new_group_name_{uuid4()}"
        change_name_response = self.sender.wakuext_service.change_group_chat_name(group_id, new_group_name)
        self.sender.verify_json_schema(change_name_response, method="wakuext_changeGroupChatName")

        self.get_message_by_content_type(
            change_name_response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=f"@{self.sender.public_key} changed the group's name to {new_group_name}",
        )[0]

        chats = change_name_response.get("result", {}).get("chats", [])
        assert chats[0].get("name", "") == new_group_name

    @pytest.mark.skip(reason="waiting for https://github.com/status-im/status-go/issues/6752 resolution")
    def test_get_group_chat_invitations(self, backend_factory):
        third_node = backend_factory("third_node")
        self.make_contacts(self.sender, third_node)

        create_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], f"group_{uuid4()}")
        group_id = create_response.get("result", {}).get("chats", [])[0].get("id")

        self.sender.wakuext_service.send_group_chat_invitation_request(group_id, third_node.public_key, f"Please join {uuid4()}")
        get_invitations_response = third_node.wakuext_service.get_group_chat_invitations()

        # CustomSchemaBuilder("wakuext_getGroupChatInvitations").create_schema(get_invitations_response)
        self.sender.verify_json_schema(get_invitations_response, method="wakuext_getGroupChatInvitations")

    def test_send_group_chat_invitation_rejection(self, backend_factory):
        third_node = backend_factory("third_node")
        self.make_contacts(self.sender, third_node)

        create_response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], f"group_{uuid4()}")
        group_id = create_response.get("result", {}).get("chats", [])[0].get("id")

        invitation_message = f"Please join {uuid4()}"
        send_invitation_response = self.sender.wakuext_service.send_group_chat_invitation_request(group_id, third_node.public_key, invitation_message)

        invitation_id = send_invitation_response.get("result", {}).get("invitations", [])[0].get("id")
        reject_response = self.sender.wakuext_service.send_group_chat_invitation_rejection(invitation_id)
        self.sender.verify_json_schema(reject_response, method="wakuext_sendGroupChatInvitationRejection")

        invitations = reject_response.get("result", {}).get("invitations", [])
        assert len(invitations) == 1
        assert invitations[0].get("chatId", "") == group_id
        assert invitations[0].get("from", "") == self.sender.public_key
        assert invitations[0].get("introductionMessage", "") == invitation_message
        assert invitations[0].get("state", 0) == 2
