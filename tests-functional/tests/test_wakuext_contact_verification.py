from enum import Enum
import pytest
from uuid import uuid4

from steps.messenger import MessengerSteps
from clients.signals import SignalType
from resources.enums import MessageContentType


class ContactVerificationState(Enum):
    ContactVerificationStatePending = 1
    ContactVerificationStateAccepted = 2
    ContactVerificationStateDeclined = 3
    ContactVerificationStateCanceled = 4


@pytest.mark.rpc
class TestContactVerification(MessengerSteps):
    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        """Initialize two backends (creator and member) for each test function"""
        self.creator = backend_new_profile("creator")
        self.member = backend_new_profile("member")
        self.fake_address = "0x" + str(uuid4())[:8]
        self.make_contacts(sender=self.creator, receiver=self.member)

    def test_send_and_accept_contact_verification_request(self):
        challenge = f"verify-accept_{uuid4()}"
        send_resp = self.creator.wakuext_service.send_contact_verification_request(self.member.public_key, challenge)
        assert send_resp.get("messages")[0].get("contentType") == MessageContentType.IDENTITY_VERIFICATION.value
        assert send_resp.get("verificationRequests")[0].get("challenge") == challenge
        assert send_resp.get("verificationRequests")[0].get("verification_status") == ContactVerificationState.ContactVerificationStatePending.value

        self.member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=challenge, timeout=10)

        get_received_resp_before = self.member.wakuext_service.get_received_verification_requests()
        req_id = get_received_resp_before[0].get("id")
        assert get_received_resp_before[0].get("challenge") == challenge
        assert get_received_resp_before[0].get("verification_status") == ContactVerificationState.ContactVerificationStatePending.value

        get_verification_resp_before = self.creator.wakuext_service.get_verification_request_sent_to(self.member.public_key)
        assert get_verification_resp_before == get_received_resp_before[0]

        get_latest_request_resp_before = self.member.wakuext_service.get_latest_verification_request_from(self.creator.public_key)
        assert get_latest_request_resp_before == get_received_resp_before[0]

        # accept the verification request as the member
        challenge_response = f"accepted-{uuid4()}"
        accept_resp = self.member.wakuext_service.accept_contact_verification_request(req_id, challenge_response)
        assert accept_resp.get("verificationRequests")[0].get("challenge") == challenge
        assert accept_resp.get("verificationRequests")[0].get("response") == challenge_response
        assert (
            accept_resp.get("verificationRequests")[0].get("verification_status") == ContactVerificationState.ContactVerificationStateAccepted.value
        )

        self.creator.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=challenge_response, timeout=10)

        get_received_resp_after = self.member.wakuext_service.get_received_verification_requests()
        assert get_received_resp_after[0].get("challenge") == challenge
        assert get_received_resp_after[0].get("response") == challenge_response
        assert get_received_resp_after[0].get("verification_status") == ContactVerificationState.ContactVerificationStateAccepted.value

        self.creator.wakuext_service.get_verification_request_sent_to(self.member.public_key)
        # assert get_verification_resp_after == get_received_resp_after[0] # https://github.com/status-im/status-go/issues/6995

        get_latest_request_resp_after = self.member.wakuext_service.get_latest_verification_request_from(self.creator.public_key)
        assert get_latest_request_resp_after == get_received_resp_after[0]

    def test_send_and_decline_contact_verification_request(self):
        challenge = f"verify-decline-{uuid4()}"
        self.creator.wakuext_service.send_contact_verification_request(self.member.public_key, challenge)

        self.member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=challenge, timeout=10)

        get_received_resp_before = self.member.wakuext_service.get_received_verification_requests()
        req_id = get_received_resp_before[0].get("id")
        assert get_received_resp_before[0].get("challenge") == challenge
        assert get_received_resp_before[0].get("verification_status") == ContactVerificationState.ContactVerificationStatePending.value

        # decline the verification request as the member
        decline_resp = self.member.wakuext_service.decline_contact_verification_request(req_id)
        assert decline_resp.get("verificationRequests")[0].get("challenge") == challenge
        assert decline_resp.get("verificationRequests")[0].get("response") == ""
        assert (
            decline_resp.get("verificationRequests")[0].get("verification_status") == ContactVerificationState.ContactVerificationStateDeclined.value
        )

        get_received_resp_after = self.member.wakuext_service.get_received_verification_requests()
        assert get_received_resp_after[0].get("challenge") == challenge
        assert get_received_resp_after[0].get("response") == ""
        assert get_received_resp_after[0].get("verification_status") == ContactVerificationState.ContactVerificationStateDeclined.value

    def test_send_and_cancel_contact_verification_request(self):
        challenge = f"verify-cancel-{uuid4()}"
        self.creator.wakuext_service.send_contact_verification_request(self.member.public_key, challenge)

        self.member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=challenge, timeout=10)

        get_received_resp_before = self.member.wakuext_service.get_received_verification_requests()
        req_id = get_received_resp_before[0].get("id")
        assert get_received_resp_before[0].get("challenge") == challenge
        assert get_received_resp_before[0].get("verification_status") == ContactVerificationState.ContactVerificationStatePending.value

        # cancel the verification request as the creator
        decline_resp = self.creator.wakuext_service.cancel_verification_request(req_id)
        assert decline_resp.get("verificationRequests")[0].get("challenge") == challenge
        assert decline_resp.get("verificationRequests")[0].get("response") == ""
        assert (
            decline_resp.get("verificationRequests")[0].get("verification_status") == ContactVerificationState.ContactVerificationStateCanceled.value
        )

        self.member.wait_for_signal_predicate(
            SignalType.MESSAGES_NEW.value,
            lambda signal: signal.get("event").get("verificationRequests")[0].get("verification_status")
            == ContactVerificationState.ContactVerificationStateCanceled.value,
        )

        get_received_resp_after = self.member.wakuext_service.get_received_verification_requests()
        assert get_received_resp_after[0].get("challenge") == challenge
        assert get_received_resp_after[0].get("response") == ""
        assert get_received_resp_after[0].get("verification_status") == ContactVerificationState.ContactVerificationStateCanceled.value
