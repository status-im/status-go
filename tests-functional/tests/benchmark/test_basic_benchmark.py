import pytest
import time

from clients.signals import SignalType
from clients.status_backend import StatusBackend
from utils.config import Config
from steps.messenger import MessengerSteps


@pytest.mark.benchmark
@pytest.mark.parametrize("waku_light_client", [False, True], indirect=True, ids=["waku_light_client_False", "waku_light_client_True"])
class TestBasicBenchmark(MessengerSteps):
    """Test StatusBackend performance.

    This test suite contains a few test cases that track CPU, RAM and network usage via Docker API.
    """

    await_signals = [
        SignalType.MESSAGES_NEW.value,
        SignalType.MESSAGE_DELIVERED.value,
        SignalType.NODE_LOGIN.value,
        SignalType.NODE_LOGOUT.value,
    ]

    @pytest.fixture(scope="function", autouse=False)
    def aut(self, request, backend_factory, waku_light_client):
        """Creates a AUT (Application Under Test) - a StatusBackend instance initialized for performance testing."""

        # Create a new user
        status_backend = backend_factory("AUT", pprof_enabled=True)
        status_backend.start_performance_monitoring()
        self.sleep(status_backend, 5)  # Keep idle for a short time

        status_backend.init_status_backend()
        status_backend.events.append("CreateAccountAndLogin")
        status_backend.create_account_and_login(waku_light_client=waku_light_client)
        status_backend.wait_for_login()
        status_backend.events.append("Logged in")
        status_backend.wakuext_service.start_messenger()

        yield status_backend

        status_backend.events.append("FreeOSMemory")
        status_backend.free_os_memory()
        time.sleep(20)

        # Finalize
        test_name = request.node.name
        filename = f"{Config.benchmark_results_dir}/{test_name}-{time.strftime('%Y%m%d-%H%M%S')}"

        metrics = status_backend.gather_metrics()
        metrics.save_to_file(f"{filename}.json")
        metrics.save_performance_chart(test_name, f"{filename}.png")
        metrics.save_go_metrics_chart(f"{test_name} - Go Metrics", f"{filename}_go_metrics.png")

    def sleep(self, backend: StatusBackend, duration: int):
        backend.events.append(f"Sleep for {duration} seconds")
        time.sleep(duration)

    def test_idle(self, aut):
        """Benchmark StatusBackend in idle state after creating a new profile"""
        self.sleep(aut, 60)

    def test_one_to_one_messages(self, aut, backend_new_profile):
        """Benchmark StatusBackend during receiving 1-1 messages."""

        # Create a message sender
        sender = backend_new_profile("sender")
        self.make_contacts(sender, aut)

        messages_count = 200
        post_sleep_duration = 30

        # Benchmark receiving many messages.
        # Don't verify reception, because it might be 1-2 messages flaky at the moment.
        aut.events.append(f"Send {messages_count} messages to AUT")
        self.send_multiple_one_to_one_messages(messages_count, sender=sender, receiver=aut)
        self.sleep(aut, post_sleep_duration)

        # Benchmark sending many messages
        aut.events.append(f"Send {messages_count} messages from AUT")
        self.send_multiple_one_to_one_messages(messages_count, sender=aut, receiver=sender)
        self.sleep(aut, post_sleep_duration)

        # Extra sleep after the test
        self.sleep(aut, 60)
