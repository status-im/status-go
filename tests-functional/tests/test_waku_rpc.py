import random
import pytest
from test_cases import StatusBackendTestCase, MessengerTestCase


class TestRpc(StatusBackendTestCase):

    @pytest.mark.parametrize(
        "method, params",
        [
            ("wakuext_peers", []),
            (
                "wakuext_activityCenterNotifications",
                [{"cursor": "", "limit": 20, "activityTypes": [5], "readType": 2}],
            ),
        ],
    )
    def test_(self, method, params):
        _id = str(random.randint(1, 8888))

        response = self.rpc_client.rpc_valid_request(method, params, _id)
        self.rpc_client.verify_json_schema(response.json(), method)


@pytest.mark.rpc
@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
class TestDefaultMessaging(MessengerTestCase):

    def test_one_to_one_messages(self):
        self.one_to_one_message(5)

    def test_add_contact(self):
        self.add_contact(execution_number=1, network_condition=None, privileged=False)

    def test_create_private_group(self):
        self.make_contacts()
        self.create_private_group(1)

    def test_private_group_messages(self):
        self.make_contacts()
        self.private_group_id = self.join_private_group()
        self.private_group_message(5, self.private_group_id)


@pytest.mark.rpc
@pytest.mark.skip
@pytest.mark.usefixtures("setup_two_privileged_nodes")
class TestLightClientMessaging(TestDefaultMessaging):

    @pytest.fixture(scope="function", autouse=False)
    def setup_two_unprivileged_nodes(self, request):
        request.cls.sender = self.sender = self.initialize_backend(self.await_signals, False)
        request.cls.receiver = self.receiver = self.initialize_backend(self.await_signals, False)
        for user in self.sender, self.receiver:
            key_uid = user.node_login_event["event"]["account"]["key-uid"]
            user.wakuext_service.set_light_client(True)
            user.logout()
            user.wait_for_logout()
            user.login(key_uid)
            user.prepare_wait_for_signal("node.login", 1)
            user.wait_for_login()
