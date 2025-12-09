from time import sleep
import pytest
from steps.messenger import MessengerSteps


@pytest.mark.reliability
class TestJoinLeaveCommunities(MessengerSteps):

    @pytest.fixture()
    def community_admin(self, backend_new_profile):
        return backend_new_profile("community_admin", bridge_network=True)

    @pytest.fixture()
    def community_member(self, backend_new_profile):
        return backend_new_profile("community_member", bridge_network=True)

    def _run_join_leave_community_baseline(self, community_admin, community_member, num_joins=1, network_condition=None):
        nodes_list = [community_admin, community_member]
        self.create_community(community_admin)
        self.leave_the_community(community_admin)

        if network_condition:
            for node in nodes_list:
                network_condition(node)

        for _ in range(num_joins):
            self.join_community(member=community_member, admin=community_admin)
            self.check_node_joined_community(community_member, joined=True)
            self.leave_the_community(community_member)
            self.check_node_joined_community(community_member, joined=False)

    @pytest.mark.skip(reason="Skipping due to failing on local build")
    # TODO: check in nightly build locally and recheck test logic
    def test_multiple_join_leave_community_requests(self, community_admin, community_member):
        self._run_join_leave_community_baseline(community_admin, community_member, num_joins=10)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_join_leave_community_with_latency(self, community_admin, community_member):
        self._run_join_leave_community_baseline(community_admin, community_member, network_condition=self.add_latency)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_join_leave_community_with_packet_loss(self, community_admin, community_member):
        self._run_join_leave_community_baseline(community_admin, community_member, network_condition=self.add_packet_loss)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_join_leave_community_with_low_bandwidth(self, community_admin, community_member):
        self._run_join_leave_community_baseline(community_admin, community_member, network_condition=self.add_low_bandwith)

    def test_join_leave_community_with_node_pause(self, community_admin, community_member):
        self.create_community(community_admin)
        self.join_community(member=community_member, admin=community_admin)
        self.check_node_joined_community(community_member, joined=True)

        with self.node_pause(community_member):
            sleep(2)
        self.leave_the_community(community_member)
        self.check_node_joined_community(community_member, joined=False)

    def test_join_leave_community_with_ip_change(self, community_admin, community_member):
        self.create_community(community_admin)
        self.join_community(member=community_member, admin=community_admin)
        self.check_node_joined_community(community_member, joined=True)

        community_member.change_container_ip()
        self.leave_the_community(community_member)
        self.check_node_joined_community(community_member, joined=False)
