from time import sleep
import pytest
from tests.test_cases import MessengerTestCase


@pytest.mark.usefixtures("setup_two_privileged_nodes")
@pytest.mark.reliability
class TestJoinLeaveCommunities(MessengerTestCase):

    def test_join_leave_community_baseline(self, num_joins=1, network_condition=None):
        nodes_list = [self.sender, self.receiver]
        self.create_community(self.sender)
        self.leave_the_community(self.sender)

        if network_condition:
            for node in nodes_list:
                network_condition(node)

        for _ in range(num_joins):
            for node in nodes_list:
                self.join_community(node)
                self.check_node_joined_community(node, joined=True)
                self.leave_the_community(node)
                self.check_node_joined_community(node, joined=False)

    def test_multiple_join_leave_community_requests(self):
        self.test_join_leave_community_baseline(num_joins=10)

    def test_join_leave_community_with_latency(self):
        self.test_join_leave_community_baseline(network_condition=self.add_latency)

    def test_join_leave_community_with_packet_loss(self):
        self.test_join_leave_community_baseline(network_condition=self.add_packet_loss)

    def test_join_leave_community_with_low_bandwidth(self):
        self.test_join_leave_community_baseline(network_condition=self.add_low_bandwith)

    def test_join_leave_community_with_node_pause(self):
        self.create_community(self.sender)
        self.join_community(self.receiver)
        self.check_node_joined_community(self.receiver, joined=True)

        with self.node_pause(self.receiver):
            sleep(2)
        self.leave_the_community(self.receiver)
        self.check_node_joined_community(self.receiver, joined=False)

    def test_join_leave_community_with_ip_change(self):
        self.create_community(self.sender)
        self.join_community(self.receiver)
        self.check_node_joined_community(self.receiver, joined=True)

        self.receiver.change_container_ip()
        self.leave_the_community(self.receiver)
        self.check_node_joined_community(self.receiver, joined=False)
