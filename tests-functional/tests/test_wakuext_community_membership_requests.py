from uuid import uuid4

import pytest

from clients.signals import SignalType
from resources.enums import RequestToJoinState
from steps.messenger import MessengerSteps
from utils.retry_utils import retry_call


@pytest.mark.rpc
class TestCommunityMembershipRequests(MessengerSteps):
    @pytest.fixture()
    def creator(self, backend_new_profile):
        return backend_new_profile("creator")

    @pytest.fixture()
    def requester(self, backend_new_profile, creator):
        req = backend_new_profile("requester")
        self.community_id = self.create_community(creator)
        self.fetch_community(req)
        return req

    @pytest.fixture()
    def fake_address(self):
        return "0x" + str(uuid4())[:8]

    def test_pending_request_to_join_community_and_cancel(self, creator, requester, fake_address):
        req_resp = requester.wakuext_service.request_to_join_community(self.community_id, [fake_address])
        assert len(req_resp.get("requestsToJoinCommunity")) == 1
        assert req_resp.get("requestsToJoinCommunity")[0].get("state") == RequestToJoinState.RequestToJoinStatePending.value
        req_id = req_resp.get("requestsToJoinCommunity")[0].get("id")

        # Requester checks his pending requests
        my_pending = requester.wakuext_service.my_pending_requests_to_join()
        assert len(my_pending) == 1
        assert my_pending[0].get("id") == req_id
        assert my_pending[0].get("communityId") == self.community_id

        # Creator fetches pending requests for community and expects at least one entry
        pending = retry_call(creator.wakuext_service.pending_requests_to_join_for_community, self.community_id)
        assert len(pending) == 1
        assert pending[0].get("id") == req_id
        assert pending[0].get("communityId") == self.community_id

        latest = requester.wakuext_service.latest_request_to_join_for_community(self.community_id)
        assert latest.get("communityId") == self.community_id
        assert latest.get("id") == req_id

        # Requester decides to cancel the request to join the community
        cancel_resp = requester.wakuext_service.cancel_request_to_join_community(latest.get("id"))
        assert len(cancel_resp.get("requestsToJoinCommunity")) == 1
        assert cancel_resp.get("requestsToJoinCommunity")[0].get("state") == RequestToJoinState.RequestToJoinStateCanceled.value

        creator.wait_for_signal_predicate(
            SignalType.MESSAGES_NEW.value,
            lambda signal: signal.get("event", {}).get("requestsToJoinCommunity")[0].get("state")
            == RequestToJoinState.RequestToJoinStateCanceled.value,
        )

        canceled = creator.wakuext_service.canceled_requests_to_join_for_community(self.community_id)
        assert len(canceled) == 1
        assert canceled[0].get("communityId") == self.community_id
        assert canceled[0].get("id") == req_id

        my_canceled = requester.wakuext_service.my_canceled_requests_to_join()
        assert len(my_canceled) == 1
        assert my_canceled[0].get("communityId") == self.community_id
        assert my_canceled[0].get("id") == req_id

        my_pending = requester.wakuext_service.my_pending_requests_to_join()
        assert my_pending is None

        pending = creator.wakuext_service.pending_requests_to_join_for_community(self.community_id)
        assert pending is None

    def test_pending_request_to_join_community_and_accept(self, creator, requester, fake_address):
        req_resp = requester.wakuext_service.request_to_join_community(self.community_id, [fake_address])
        assert len(req_resp.get("requestsToJoinCommunity")) == 1
        assert req_resp.get("requestsToJoinCommunity")[0].get("state") == RequestToJoinState.RequestToJoinStatePending.value
        req_id = req_resp.get("requestsToJoinCommunity")[0].get("id")

        # Requester checks his pending requests
        my_pending = requester.wakuext_service.my_pending_requests_to_join()
        assert len(my_pending) == 1
        assert my_pending[0].get("id") == req_id
        assert my_pending[0].get("communityId") == self.community_id

        # Creator fetches pending requests for community and expects at least one entry
        pending = retry_call(creator.wakuext_service.pending_requests_to_join_for_community, self.community_id)
        assert len(pending) == 1
        assert pending[0].get("id") == req_id
        assert pending[0].get("communityId") == self.community_id

        latest = requester.wakuext_service.latest_request_to_join_for_community(self.community_id)
        assert latest.get("communityId") == self.community_id
        assert latest.get("id") == req_id

        all_non_approved = creator.wakuext_service.all_non_approved_communities_requests_to_join()
        assert len(all_non_approved) == 1
        assert all_non_approved[0].get("communityId") == self.community_id
        assert all_non_approved[0].get("id") == req_id

        # Creator accepts the request from the requester to join the community
        accept_resp = creator.wakuext_service.accept_request_to_join_community(latest.get("id"))
        assert len(accept_resp.get("requestsToJoinCommunity")) == 1
        assert accept_resp.get("requestsToJoinCommunity")[0].get("state") == RequestToJoinState.RequestToJoinStateAccepted.value

        requester.wait_for_signal_predicate(
            SignalType.MESSAGES_NEW.value,
            lambda signal: signal.get("event", {}).get("requestsToJoinCommunity")[0].get("state")
            == RequestToJoinState.RequestToJoinStateAccepted.value,
        )

        canceled = creator.wakuext_service.canceled_requests_to_join_for_community(self.community_id)
        assert canceled is None

        my_canceled = requester.wakuext_service.my_canceled_requests_to_join()
        assert my_canceled is None

        my_pending = requester.wakuext_service.my_pending_requests_to_join()
        assert my_pending is None

        pending = creator.wakuext_service.pending_requests_to_join_for_community(self.community_id)
        assert pending is None

        all_non_approved = creator.wakuext_service.all_non_approved_communities_requests_to_join()
        assert all_non_approved == []

    def test_pending_request_to_join_community_and_decline(self, creator, requester, fake_address):
        req_resp = requester.wakuext_service.request_to_join_community(self.community_id, [fake_address])
        assert len(req_resp.get("requestsToJoinCommunity")) == 1
        assert req_resp.get("requestsToJoinCommunity")[0].get("state") == RequestToJoinState.RequestToJoinStatePending.value
        req_id = req_resp.get("requestsToJoinCommunity")[0].get("id")

        # Requester checks his pending requests
        my_pending = requester.wakuext_service.my_pending_requests_to_join()
        assert len(my_pending) == 1
        assert my_pending[0].get("id") == req_id
        assert my_pending[0].get("communityId") == self.community_id

        # Creator fetches pending requests for community and expects at least one entry
        pending = retry_call(creator.wakuext_service.pending_requests_to_join_for_community, self.community_id)
        assert len(pending) == 1
        assert pending[0].get("id") == req_id
        assert pending[0].get("communityId") == self.community_id

        latest = requester.wakuext_service.latest_request_to_join_for_community(self.community_id)
        assert latest.get("communityId") == self.community_id
        assert latest.get("id") == req_id

        # Creator declines the request from the requester to join the community
        decline_resp = creator.wakuext_service.decline_request_to_join_community(latest.get("id"))
        assert decline_resp.get("requestsToJoinCommunity")[0].get("state") == RequestToJoinState.RequestToJoinStateDeclined.value

        canceled = creator.wakuext_service.canceled_requests_to_join_for_community(self.community_id)
        assert canceled is None

        declined = creator.wakuext_service.declined_requests_to_join_for_community(self.community_id)
        assert len(declined) == 1
        assert declined[0].get("communityId") == self.community_id
        assert declined[0].get("id") == req_id

        my_canceled = requester.wakuext_service.my_canceled_requests_to_join()
        assert my_canceled is None

        my_pending = requester.wakuext_service.my_pending_requests_to_join()
        # assert my_pending is None # bug reported here https://github.com/status-im/status-go/issues/6975

        pending = creator.wakuext_service.pending_requests_to_join_for_community(self.community_id)
        assert pending is None

    def test_check_and_delete_pending_request_to_join_community(self, creator, requester, fake_address):
        req_resp = requester.wakuext_service.request_to_join_community(self.community_id, [fake_address])
        assert len(req_resp.get("requestsToJoinCommunity")) == 1
        req_id = req_resp.get("requestsToJoinCommunity")[0].get("id")

        # Requester checks his pending requests
        my_pending = requester.wakuext_service.my_pending_requests_to_join()
        assert len(my_pending) == 1
        assert my_pending[0].get("id") == req_id
        assert my_pending[0].get("communityId") == self.community_id

        retry_call(creator.wakuext_service.pending_requests_to_join_for_community, self.community_id)

        requester.wakuext_service.check_and_delete_pending_request_to_join_community()
        creator.wakuext_service.check_and_delete_pending_request_to_join_community()

        my_pending = requester.wakuext_service.my_pending_requests_to_join()
        # assert my_pending is None # bug reported here https://github.com/status-im/status-go/issues/6976

        creator.wakuext_service.pending_requests_to_join_for_community(self.community_id)
        # assert pending is None # bug reported here https://github.com/status-im/status-go/issues/6976

    def test_check_permissions_and_generate_community_requests(self, creator, requester):
        perm_resp = requester.wakuext_service.check_permissions_to_join_community(self.community_id)
        assert perm_resp.get("satisfied") is True
        assert len(perm_resp.get("roles")) == 1
        assert perm_resp.get("roles")[0].get("type") == 2

        member_pub_key = requester.public_key

        gen_join_resp = requester.wakuext_service.generate_joining_community_requests_for_signing(member_pub_key, self.community_id, [])
        assert len(gen_join_resp) == 1
        assert gen_join_resp[0].get("account").lower() == perm_resp.get("validCombinations")[0].get("address").lower()

        gen_edit_resp = requester.wakuext_service.generate_edit_community_requests_for_signing(member_pub_key, self.community_id, [])
        assert len(gen_edit_resp) == 1
        assert gen_edit_resp[0].get("account").lower() == perm_resp.get("validCombinations")[0].get("address").lower()
