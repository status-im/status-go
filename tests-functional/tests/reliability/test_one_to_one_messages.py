from time import sleep
from uuid import uuid4
import pytest
from steps import messenger
from clients.signals import SignalType
from resources.constants import USE_IPV6


@pytest.mark.reliability
class TestOneToOneMessages:

    @pytest.fixture()
    def sender(self, backend_new_profile):
        return backend_new_profile("sender", bridge_network=True)

    @pytest.fixture()
    def receiver(self, backend_new_profile):
        return backend_new_profile("receiver", bridge_network=True)

    def _run_one_to_one_message_baseline(self, sender, receiver, message_count=1):
        messenger.one_to_one_message(message_count, sender=sender, receiver=receiver)

    def test_one_to_one_message_baseline(self, sender, receiver, message_count=1):
        self._run_one_to_one_message_baseline(sender, receiver, message_count)

    def test_multiple_one_to_one_messages(self, sender, receiver):
        self._run_one_to_one_message_baseline(sender, receiver, message_count=50)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_one_to_one_message_with_latency(self, sender, receiver):
        with messenger.add_latency(receiver):
            self._run_one_to_one_message_baseline(sender, receiver, message_count=50)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_one_to_one_message_with_packet_loss(self, sender, receiver):
        with messenger.add_packet_loss(receiver):
            self._run_one_to_one_message_baseline(sender, receiver, message_count=50)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_one_to_one_message_with_low_bandwidth(self, sender, receiver):
        with messenger.add_low_bandwith(receiver):
            self._run_one_to_one_message_baseline(sender, receiver, message_count=50)

    def test_one_to_one_message_with_node_pause_30_seconds(self, sender, receiver):
        with messenger.node_pause(receiver):
            message_text = f"test_message_{uuid4()}"
            sender.wakuext_service.send_one_to_one_message(receiver.public_key, message_text)
            sleep(30)
        with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=message_text):
            pass
        with sender.expect_signal(SignalType.MESSAGE_DELIVERED):
            pass

    @pytest.mark.skipif(USE_IPV6 == "Yes", reason="Test works only with IPV4")
    def test_one_to_one_messages_with_ip_change(self, sender, receiver):
        self._run_one_to_one_message_baseline(sender, receiver)
        receiver.change_container_ip()
        self._run_one_to_one_message_baseline(sender, receiver)
