from time import sleep
import pytest
from steps.messenger import MessengerSteps


@pytest.mark.reliability
class TestJoinLeaveCommunities(MessengerSteps):

    @pytest.fixture()
    def admin(self, backend_new_profile):
        return backend_new_profile("admin", bridge_network=True)

    @pytest.fixture()
    def member(self, backend_new_profile):
        return backend_new_profile("member", bridge_network=True)

    def _run_join_leave_community_baseline(self, admin, member, num_joins=1, network_condition=None):
        nodes_list = [admin, member]
        self.create_community(admin)
        self.leave_the_community(admin)

        if network_condition:
            for node in nodes_list:
                network_condition(node)

        for _ in range(num_joins):
            self.join_community(member=member, admin=admin)
            self.check_node_joined_community(member, joined=True)
            self.leave_the_community(member)
            self.check_node_joined_community(member, joined=False)

    @pytest.mark.skip(reason="Skipping due to failing on local build")
    # TODO: check in nightly build locally and recheck test logic
    def test_multiple_join_leave_community_requests(self, admin, member):
        self._run_join_leave_community_baseline(admin, member, num_joins=10)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_join_leave_community_with_latency(self, admin, member):
        self._run_join_leave_community_baseline(admin, member, network_condition=self.add_latency)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_join_leave_community_with_packet_loss(self, admin, member):
        self._run_join_leave_community_baseline(admin, member, network_condition=self.add_packet_loss)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_join_leave_community_with_low_bandwidth(self, admin, member):
        self._run_join_leave_community_baseline(admin, member, network_condition=self.add_low_bandwith)

    def test_join_leave_community_with_node_pause(self, admin, member):
        self.create_community(admin)
        self.join_community(member=member, admin=admin)
        self.check_node_joined_community(member, joined=True)

        with self.node_pause(member):
            sleep(2)
        self.leave_the_community(member)
        self.check_node_joined_community(member, joined=False)

    def test_join_leave_community_with_ip_change(self, admin, member):
        self.create_community(admin)
        self.join_community(member=member, admin=admin)
        self.check_node_joined_community(member, joined=True)

        member.change_container_ip()
        self.leave_the_community(member)
        self.check_node_joined_community(member, joined=False)
