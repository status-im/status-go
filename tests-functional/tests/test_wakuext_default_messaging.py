import logging
import os

import pytest
import tempfile

from steps.messenger import MessengerSteps
from clients.status_backend import StatusBackend

@pytest.mark.rpc
@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
class TestDefaultMessaging(MessengerSteps):

    def test_one_to_one_messages(self):
        responses = self.one_to_one_message(5)

        for response in responses:
            self.receiver.verify_json_schema(response, method="wakuext_sendOneToOneMessage")

            chat = response["result"]["chats"][0]
            assert chat["id"] == self.receiver.public_key
            assert chat["lastMessage"]["displayName"] == self.sender.display_name

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


@pytest.fixture
def temp_dir():
    with tempfile.TemporaryDirectory() as tempdir:
        yield tempdir


@pytest.mark.rpc
class TestLocalWakuFleet(MessengerSteps):
    def create_backend(self, data_dir, url, wakuV2LightClient=False):
        backend = StatusBackend(self.await_signals, status_backend_url=url)
        try:
            backend.logout()
        except Exception:
            logging.warning("failed to logout from %s", url)
        backend.init_status_backend(data_dir)
        backend.create_account_and_login(data_dir, wakuV2Fleet="sirotin.dev", wakuV2LightClient=wakuV2LightClient)
        # backend.create_account_and_login(data_dir, wakuV2Fleet="status.prod", wakuV2LightClient=wakuV2LightClient)
        backend.find_public_key()
        backend.wakuext_service.start_messenger()
        backend.wait_for_online()
        return backend

    def test_lightpush(self, temp_dir):
        dir_1 = os.path.join(temp_dir, "1")
        self.sender = self.create_backend(dir_1, wakuV2LightClient=True)
        self.sender.wakuext_service.send_message()

    def test_contact_request(self, temp_dir):
        dir_1 = os.path.join(temp_dir, "1")
        dir_2 = os.path.join(temp_dir, "2")
        self.sender = self.create_backend(dir_1, "http://localhost:12345", wakuV2LightClient=True)
        self.receiver = self.create_backend(dir_2, "http://localhost:12346", wakuV2LightClient=False)

        # publishing message via lightpush
        print(f"keeping status-backend running {temp_dir}")
        input("Press Enter to continue to make contacts...")

        try:
            self.make_contacts()
        except Exception as e:
            logging.error(f"❌failed to make contacts: {e}")
            print(f"keeping status-backend running {temp_dir}")
            input("Press Enter to continue...")

        self.sender.logout()
        self.receiver.logout()

    def test_community(self, request, temp_dir):
        dir_1 = os.path.join(temp_dir, "1")
        dir_2 = os.path.join(temp_dir, "2")
        dir_3 = os.path.join(temp_dir, "3")

        owner = self.create_backend(dir_1, "http://localhost:12345")
        desktop = self.create_backend(dir_2, "http://localhost:12346", wakuV2LightClient=False)
        mobile = self.create_backend(dir_3, "http://localhost:12347", wakuV2LightClient=True)

        self.sender = owner

        self.community_id = self.create_community(owner)
        logging.info(f"✅ community created {self.community_id}")

        try:
            self.join_community(desktop)
            logging.info(f"✅community joined on DESKTOP")
            self.join_community(mobile)
            logging.info(f"✅community joined on MOBILE")
        except Exception as e:
            logging.error(f"❌failed to join community: {e}")

            print("keeping status-backend running")
            print(f"status-backend-1: {dir_1}")
            print(f"status-backend-2: {dir_2}")
            print(f"status-backend-3: {dir_3}")

            input("Press Enter to continue...")

        owner.logout()
        desktop.logout()
        mobile.logout()