from uuid import uuid4
import pytest
from steps.messenger import MessengerSteps
from resources.enums import MessageContentType


@pytest.mark.reliability
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
        assert "error" not in leave_group_response

    def test_create_group_chat_from_invitation(self):
        self.make_contacts(self.sender, self.receiver)
        group_name = f"invitation_group_{uuid4()}"
        create_resp = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], group_name)
        group_id = create_resp.get("result", {}).get("chat", {}).get("id")
        assert group_id is not None
        # Simulate invitation by sending invitation request
        invite_resp = self.sender.wakuext_service.send_group_chat_invitation_request(group_id, self.receiver.public_key, "Please join")
        assert "error" not in invite_resp
        # Receiver creates group chat from invitation
        create_from_inv_resp = self.receiver.wakuext_service.create_group_chat_from_invitation(group_name, group_id, self.sender.public_key)
        assert "error" not in create_from_inv_resp

    def test_add_members_to_group_chat(self):
        self.make_contacts(self.sender, self.receiver)
        group_name = f"add_members_group_{uuid4()}"
        create_resp = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], group_name)
        group_id = create_resp.get("result", {}).get("chat", {}).get("id")
        assert group_id is not None
        # Add sender again as member (for test)
        add_resp = self.sender.wakuext_service.add_members_to_group_chat(group_id, [self.sender.public_key])
        assert "error" not in add_resp

    def test_remove_member_from_group_chat(self):
        self.make_contacts(self.sender, self.receiver)
        group_name = f"remove_member_group_{uuid4()}"
        create_resp = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key, self.sender.public_key], group_name)
        group_id = create_resp.get("result", {}).get("chat", {}).get("id")
        assert group_id is not None
        remove_resp = self.sender.wakuext_service.remove_member_from_group_chat(group_id, self.sender.public_key)
        assert "error" not in remove_resp

    def test_remove_members_from_group_chat(self):
        self.make_contacts(self.sender, self.receiver)
        group_name = f"remove_members_group_{uuid4()}"
        create_resp = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key, self.sender.public_key], group_name)
        group_id = create_resp.get("result", {}).get("chat", {}).get("id")
        assert group_id is not None
        remove_resp = self.sender.wakuext_service.remove_members_from_group_chat(group_id, [self.sender.public_key, self.receiver.public_key])
        assert "error" not in remove_resp

    def test_confirm_joining_group(self):
        self.make_contacts(self.sender, self.receiver)
        group_name = f"confirm_join_group_{uuid4()}"
        create_resp = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], group_name)
        group_id = create_resp.get("result", {}).get("chat", {}).get("id")
        assert group_id is not None
        confirm_resp = self.receiver.wakuext_service.confirm_joining_group(group_id)
        assert "error" not in confirm_resp

    def test_change_group_chat_name(self):
        self.make_contacts(self.sender, self.receiver)
        group_name = f"change_name_group_{uuid4()}"
        create_resp = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], group_name)
        group_id = create_resp.get("result", {}).get("chat", {}).get("id")
        assert group_id is not None
        new_name = f"new_name_{uuid4()}"
        change_resp = self.sender.wakuext_service.change_group_chat_name(group_id, new_name)
        assert "error" not in change_resp

    def test_send_group_chat_invitation_request(self):
        self.make_contacts(self.sender, self.receiver)
        group_name = f"invitation_request_group_{uuid4()}"
        create_resp = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], group_name)
        group_id = create_resp.get("result", {}).get("chat", {}).get("id")
        assert group_id is not None
        invite_resp = self.sender.wakuext_service.send_group_chat_invitation_request(group_id, self.receiver.public_key, "Join us!")
        assert "error" not in invite_resp

    def test_get_group_chat_invitations(self):
        self.make_contacts(self.sender, self.receiver)
        group_name = f"get_invitations_group_{uuid4()}"
        create_resp = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], group_name)
        group_id = create_resp.get("result", {}).get("chat", {}).get("id")
        assert group_id is not None
        self.sender.wakuext_service.send_group_chat_invitation_request(group_id, self.receiver.public_key, "Please join")
        invitations = self.receiver.wakuext_service.get_group_chat_invitations()
        assert isinstance(invitations, list)

    def test_send_group_chat_invitation_rejection(self):
        self.make_contacts(self.sender, self.receiver)
        group_name = f"reject_invitation_group_{uuid4()}"
        create_resp = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], group_name)
        group_id = create_resp.get("result", {}).get("chat", {}).get("id")
        assert group_id is not None
        self.sender.wakuext_service.send_group_chat_invitation_request(group_id, self.receiver.public_key, "Please join")
        invitations = self.receiver.wakuext_service.get_group_chat_invitations()
        assert isinstance(invitations, list) and len(invitations) > 0
        invitation_id = invitations[0].get("id") if isinstance(invitations[0], dict) else None
        assert invitation_id is not None
        reject_resp = self.receiver.wakuext_service.send_group_chat_invitation_rejection(invitation_id)
        assert "error" not in reject_resp
