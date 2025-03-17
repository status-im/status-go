import pytest
from steps.messenger import MessengerSteps


@pytest.mark.rpc
@pytest.mark.skip
@pytest.mark.usefixtures("setup_two_privileged_nodes")
class TestLightClientMessaging(MessengerSteps):

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
