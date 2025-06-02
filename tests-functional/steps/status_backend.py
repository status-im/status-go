from clients.services.wallet import WalletService
from clients.signals import SignalType
from clients.status_backend import StatusBackend
from conftest import option
from resources.constants import ANVIL_NETWORK_ID


class StatusBackendSteps:
    reuse_container = True  # Skip close_status_backend_containers cleanup
    await_signals = [SignalType.NODE_LOGIN.value]

    network_id = ANVIL_NETWORK_ID

    @classmethod
    def setup_class(cls, skip_login=False):
        cls.rpc_client = StatusBackend(await_signals=cls.await_signals)
        cls.wallet_service = WalletService(cls.rpc_client)
        cls.rpc_client.init_status_backend()

        if not skip_login:
            cls.rpc_client.restore_account_and_login()
            cls.rpc_client.wait_for_login()

    def teardown_class(self):
        for status_backend in option.status_backend_containers:
            status_backend.container.stop(timeout=10)
            option.status_backend_containers.remove(status_backend)
            status_backend.container.remove()
