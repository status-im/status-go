from clients.services.wallet import WalletService
from clients.signals import SignalType
from clients.status_backend import StatusBackend
from conftest import option


class StatusBackendSteps:

    reuse_container = True  # Skip close_status_backend_containers cleanup
    await_signals = [SignalType.NODE_LOGIN.value]

    network_id = 31337

    def setup_class(self):
        self.rpc_client = StatusBackend(await_signals=self.await_signals)
        self.wallet_service = WalletService(self.rpc_client)

        self.rpc_client.init_status_backend()
        self.rpc_client.restore_account_and_login()
        self.rpc_client.wait_for_login()

    def teardown_class(self):
        for status_backend in option.status_backend_containers:
            status_backend.container.stop(timeout=10)
            option.status_backend_containers.remove(status_backend)
            status_backend.container.remove()
