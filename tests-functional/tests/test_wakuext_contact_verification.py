from uuid import uuid4

import pytest

from clients.signals import SignalType
from resources.enums import ContactVerificationState, MessageContentType
from steps import async_messenger


@pytest.mark.rpc
@pytest.mark.asyncio
class TestContactVerification:

    @pytest.fixture
    async def creator(self, async_backend_new_profile):
        return await async_backend_new_profile("creator")

    @pytest.fixture
    async def member(self, async_backend_new_profile, creator):
        m = await async_backend_new_profile("member")
        await async_messenger.make_contacts(sender=creator, receiver=m)
        return m

    @pytest.fixture
    def fake_address(self):
        return "0x" + str(uuid4())[:8]

    async def test_send_and_accept_contact_verification_request(self, creator, member, fake_address):
        challenge = f"verify-accept_{uuid4()}"
        send_resp = creator.wakuext_service.send_contact_verification_request(member.public_key, challenge)
        assert send_resp.get("messages")[0].get("contentType") == MessageContentType.IDENTITY_VERIFICATION.value
        assert send_resp.get("verificationRequests")[0].get("challenge") == challenge
        assert send_resp.get("verificationRequests")[0].get("verification_status") == ContactVerificationState.Pending.value

        await member.wait_for_signal(SignalType.MESSAGES_NEW, pattern=challenge, timeout=10, check_buffer=True)

        get_received_resp_before = member.wakuext_service.get_received_verification_requests()
        req_id = get_received_resp_before[0].get("id")
        assert get_received_resp_before[0].get("challenge") == challenge
        assert get_received_resp_before[0].get("verification_status") == ContactVerificationState.Pending.value

        get_verification_resp_before = creator.wakuext_service.get_verification_request_sent_to(member.public_key)
        assert get_verification_resp_before == get_received_resp_before[0]

        get_latest_request_resp_before = member.wakuext_service.get_latest_verification_request_from(creator.public_key)
        assert get_latest_request_resp_before == get_received_resp_before[0]

        # accept the verification request as the member
        challenge_response = f"accepted-{uuid4()}"
        accept_resp = member.wakuext_service.accept_contact_verification_request(req_id, challenge_response)
        assert accept_resp.get("verificationRequests")[0].get("challenge") == challenge
        assert accept_resp.get("verificationRequests")[0].get("response") == challenge_response
        assert accept_resp.get("verificationRequests")[0].get("verification_status") == ContactVerificationState.Accepted.value

        await creator.wait_for_signal(SignalType.MESSAGES_NEW, pattern=challenge_response, timeout=10, check_buffer=True)

        get_received_resp_after = member.wakuext_service.get_received_verification_requests()
        assert get_received_resp_after[0].get("challenge") == challenge
        assert get_received_resp_after[0].get("response") == challenge_response
        assert get_received_resp_after[0].get("verification_status") == ContactVerificationState.Accepted.value

        creator.wakuext_service.get_verification_request_sent_to(member.public_key)
        # assert get_verification_resp_after == get_received_resp_after[0] # https://github.com/status-im/status-go/issues/6995

        get_latest_request_resp_after = member.wakuext_service.get_latest_verification_request_from(creator.public_key)
        assert get_latest_request_resp_after == get_received_resp_after[0]

    async def test_send_and_decline_contact_verification_request(self, creator, member, fake_address):
        challenge = f"verify-decline-{uuid4()}"
        creator.wakuext_service.send_contact_verification_request(member.public_key, challenge)

        await member.wait_for_signal(SignalType.MESSAGES_NEW, pattern=challenge, timeout=10, check_buffer=True)

        get_received_resp_before = member.wakuext_service.get_received_verification_requests()
        req_id = get_received_resp_before[0].get("id")
        assert get_received_resp_before[0].get("challenge") == challenge
        assert get_received_resp_before[0].get("verification_status") == ContactVerificationState.Pending.value

        # decline the verification request as the member
        decline_resp = member.wakuext_service.decline_contact_verification_request(req_id)
        assert decline_resp.get("verificationRequests")[0].get("challenge") == challenge
        assert decline_resp.get("verificationRequests")[0].get("response") == ""
        assert decline_resp.get("verificationRequests")[0].get("verification_status") == ContactVerificationState.Declined.value

        get_received_resp_after = member.wakuext_service.get_received_verification_requests()
        assert get_received_resp_after[0].get("challenge") == challenge
        assert get_received_resp_after[0].get("response") == ""
        assert get_received_resp_after[0].get("verification_status") == ContactVerificationState.Declined.value

    async def test_send_and_cancel_contact_verification_request(self, creator, member, fake_address):
        challenge = f"verify-cancel-{uuid4()}"
        creator.wakuext_service.send_contact_verification_request(member.public_key, challenge)

        await member.wait_for_signal(SignalType.MESSAGES_NEW, pattern=challenge, timeout=10, check_buffer=True)

        get_received_resp_before = member.wakuext_service.get_received_verification_requests()
        req_id = get_received_resp_before[0].get("id")
        assert get_received_resp_before[0].get("challenge") == challenge
        assert get_received_resp_before[0].get("verification_status") == ContactVerificationState.Pending.value

        # cancel the verification request as the creator
        decline_resp = creator.wakuext_service.cancel_verification_request(req_id)
        assert decline_resp.get("verificationRequests")[0].get("challenge") == challenge
        assert decline_resp.get("verificationRequests")[0].get("response") == ""
        assert decline_resp.get("verificationRequests")[0].get("verification_status") == ContactVerificationState.Canceled.value

        await member.wait_for_verification_status(challenge, ContactVerificationState.Canceled)

        get_received_resp_after = member.wakuext_service.get_received_verification_requests()
        assert get_received_resp_after[0].get("challenge") == challenge
        assert get_received_resp_after[0].get("response") == ""
        assert get_received_resp_after[0].get("verification_status") == ContactVerificationState.Canceled.value
