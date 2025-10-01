from time import sleep
from uuid import uuid4
import pytest

from steps.messenger import MessengerSteps


@pytest.mark.rpc
class TestCommunityMembershipRequests(MessengerSteps):
    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        """Initialize two backends (creator and requester) for each test function"""
        self.creator = backend_new_profile("creator")
        self.requester = backend_new_profile("requester")
        self.fake_address = "0x" + str(uuid4())[:8]
        self.community_id = self.create_community(self.creator)
        self.fetch_community(self.requester)

    def test_pending_request_to_join_community_and_cancel(self):
        req_resp = self.requester.wakuext_service.request_to_join_community(self.community_id, self.fake_address)
        assert req_resp.get("requestsToJoinCommunity")[0].get("state") == 1
        req_id = req_resp.get("requestsToJoinCommunity")[0].get("id")

        sleep(2)  # small sleep is needed for community requests to sync across nodes

        # Requester checks his pending requests
        my_pending = self.requester.wakuext_service.my_pending_requests_to_join()
        assert my_pending[0].get("id") == req_id
        assert my_pending[0].get("communityId") == self.community_id

        # Creator fetches pending requests for community and expects at least one entry
        pending = self.creator.wakuext_service.pending_requests_to_join_for_community(self.community_id)
        assert pending[0].get("id") == req_id
        assert pending[0].get("communityId") == self.community_id

        latest = self.requester.wakuext_service.latest_request_to_join_for_community(self.community_id)
        assert latest.get("communityId") == self.community_id
        assert latest.get("id") == req_id

        # Requester decides to cancel the request to join the community
        cancel_resp = self.requester.wakuext_service.cancel_request_to_join_community(latest.get("id"))
        assert cancel_resp.get("requestsToJoinCommunity")[0].get("state") == 4

        sleep(2)  # small sleep is needed for community requests to sync across nodes

        canceled = self.creator.wakuext_service.canceled_requests_to_join_for_community(self.community_id)
        assert canceled[0].get("communityId") == self.community_id
        assert canceled[0].get("id") == req_id

        my_canceled = self.requester.wakuext_service.my_canceled_requests_to_join()
        assert my_canceled[0].get("communityId") == self.community_id
        assert my_canceled[0].get("id") == req_id

        my_pending = self.requester.wakuext_service.my_pending_requests_to_join()
        assert my_pending is None

        pending = self.creator.wakuext_service.pending_requests_to_join_for_community(self.community_id)
        assert pending is None

    def test_pending_request_to_join_community_and_accept(self):
        req_resp = self.requester.wakuext_service.request_to_join_community(self.community_id, self.fake_address)
        assert req_resp.get("requestsToJoinCommunity")[0].get("state") == 1
        req_id = req_resp.get("requestsToJoinCommunity")[0].get("id")

        sleep(2)  # small sleep is needed for community requests to sync across nodes

        # Requester checks his pending requests
        my_pending = self.requester.wakuext_service.my_pending_requests_to_join()
        assert my_pending[0].get("id") == req_id
        assert my_pending[0].get("communityId") == self.community_id

        # Creator fetches pending requests for community and expects at least one entry
        pending = self.creator.wakuext_service.pending_requests_to_join_for_community(self.community_id)
        assert pending[0].get("id") == req_id
        assert pending[0].get("communityId") == self.community_id

        latest = self.requester.wakuext_service.latest_request_to_join_for_community(self.community_id)
        assert latest.get("communityId") == self.community_id
        assert latest.get("id") == req_id

        all_non_approved = self.creator.wakuext_service.all_non_approved_communities_requests_to_join()
        assert all_non_approved[0].get("communityId") == self.community_id
        assert all_non_approved[0].get("id") == req_id

        # Creator accepts the request from the requester to join the community
        cancel_resp = self.creator.wakuext_service.accept_request_to_join_community(latest.get("id"))
        assert cancel_resp.get("requestsToJoinCommunity")[0].get("state") == 3

        sleep(2)  # small sleep is needed for community requests to sync across nodes

        canceled = self.creator.wakuext_service.canceled_requests_to_join_for_community(self.community_id)
        assert canceled is None

        my_canceled = self.requester.wakuext_service.my_canceled_requests_to_join()
        assert my_canceled is None

        my_pending = self.requester.wakuext_service.my_pending_requests_to_join()
        assert my_pending is None

        pending = self.creator.wakuext_service.pending_requests_to_join_for_community(self.community_id)
        assert pending is None

        all_non_approved = self.creator.wakuext_service.all_non_approved_communities_requests_to_join()
        assert all_non_approved == []

    def test_pending_request_to_join_community_and_decline(self):
        req_resp = self.requester.wakuext_service.request_to_join_community(self.community_id, self.fake_address)
        assert req_resp.get("requestsToJoinCommunity")[0].get("state") == 1
        req_id = req_resp.get("requestsToJoinCommunity")[0].get("id")

        sleep(2)  # small sleep is needed for community requests to sync across nodes

        # Requester checks his pending requests
        my_pending = self.requester.wakuext_service.my_pending_requests_to_join()
        assert my_pending[0].get("id") == req_id
        assert my_pending[0].get("communityId") == self.community_id

        # Creator fetches pending requests for community and expects at least one entry
        pending = self.creator.wakuext_service.pending_requests_to_join_for_community(self.community_id)
        assert pending[0].get("id") == req_id
        assert pending[0].get("communityId") == self.community_id

        latest = self.requester.wakuext_service.latest_request_to_join_for_community(self.community_id)
        assert latest.get("communityId") == self.community_id
        assert latest.get("id") == req_id

        # Creator declines the request from the requester to join the community
        cancel_resp = self.creator.wakuext_service.decline_request_to_join_community(latest.get("id"))
        assert cancel_resp.get("requestsToJoinCommunity")[0].get("state") == 2

        sleep(2)  # small sleep is needed for community requests to sync across nodes

        canceled = self.creator.wakuext_service.canceled_requests_to_join_for_community(self.community_id)
        assert canceled is None

        declined = self.creator.wakuext_service.declined_requests_to_join_for_community(self.community_id)
        assert declined[0].get("communityId") == self.community_id
        assert declined[0].get("id") == req_id

        my_canceled = self.requester.wakuext_service.my_canceled_requests_to_join()
        assert my_canceled is None

        my_pending = self.requester.wakuext_service.my_pending_requests_to_join()
        # assert my_pending is None # bug reported here https://github.com/status-im/status-go/issues/6975

        pending = self.creator.wakuext_service.pending_requests_to_join_for_community(self.community_id)
        assert pending is None

    def test_check_and_delete_pending_request_to_join_community(self):
        req_resp = self.requester.wakuext_service.request_to_join_community(self.community_id, self.fake_address)
        assert req_resp.get("requestsToJoinCommunity")[0].get("state") == 1
        req_id = req_resp.get("requestsToJoinCommunity")[0].get("id")

        sleep(2)  # small sleep is needed for community requests to sync across nodes

        # Requester checks his pending requests
        my_pending = self.requester.wakuext_service.my_pending_requests_to_join()
        assert my_pending[0].get("id") == req_id
        assert my_pending[0].get("communityId") == self.community_id

        self.requester.wakuext_service.check_and_delete_pending_request_to_join_community()
        self.creator.wakuext_service.check_and_delete_pending_request_to_join_community()

        sleep(2)  # small sleep is needed for community requests to sync across nodes

        my_pending = self.requester.wakuext_service.my_pending_requests_to_join()
        # assert my_pending is None # bug reported here https://github.com/status-im/status-go/issues/6976
        self.creator.wakuext_service.pending_requests_to_join_for_community(self.community_id)
        # assert pending is None # bug reported here https://github.com/status-im/status-go/issues/6976

    def test_check_permissions_and_generate_community_requests(self):
        perm_resp = self.requester.wakuext_service.check_permissions_to_join_community(self.community_id)
        assert perm_resp.get("satisfied") is True
        assert perm_resp.get("roles")[0].get("type") == 2

        member_pub_key = self.requester.public_key

        gen_join_resp = self.requester.wakuext_service.generate_joining_community_requests_for_signing(member_pub_key, self.community_id, [])
        assert gen_join_resp[0].get("account").lower() == perm_resp.get("validCombinations")[0].get("address").lower()

        gen_edit_resp = self.requester.wakuext_service.generate_edit_community_requests_for_signing(member_pub_key, self.community_id, [])
        assert gen_edit_resp[0].get("account").lower() == perm_resp.get("validCombinations")[0].get("address").lower()
