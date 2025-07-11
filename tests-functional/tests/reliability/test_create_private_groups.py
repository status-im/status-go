from time import sleep
from uuid import uuid4
import pytest
from steps.messenger import MessengerSteps
from clients.signals import SignalType
from resources.constants import USE_IPV6


@pytest.mark.reliability
class TestCreatePrivateGroups(MessengerSteps):

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_factory):
        """Initialize two unprivileged backends (sender and receiver) for each test function"""
        self.sender = backend_factory("sender")
        self.receiver = backend_factory("receiver")

    def test_create_private_group_baseline(self, private_groups_count=1):
        self.make_contacts(self.sender, self.receiver)
        self.create_private_group(private_groups_count)

    def test_multiple_create_private_groups(self):
        self.test_create_private_group_baseline(private_groups_count=50)

    def test_create_private_groups_with_node_pause_30_seconds(self):
        self.make_contacts(self.sender, self.receiver)

        with self.node_pause(self.receiver):
            private_group_name = f"private_group_{uuid4()}"
            self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], private_group_name)
            sleep(30)
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=private_group_name)

    @pytest.mark.skipif(USE_IPV6 == "Yes", reason="Test works only with IPV4")
    def test_create_private_groups_with_ip_change(self):
        self.make_contacts(self.sender, self.receiver)
        self.receiver.change_container_ip()

        private_group_name = f"private_group_{uuid4()}"
        self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], private_group_name)
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=private_group_name)

    def test_leave_group_chat(self):
        self.make_contacts(self.sender, self.receiver)
        group_name = f"leave_group_{uuid4()}"
        response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], group_name)
        group_id = response.get("result", {}).get("chats", [])[0].get("id")
        assert group_id is not None
        leave_response = self.sender.wakuext_service.leave_group_chat(group_id, True)
        assert "error" not in leave_response

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
