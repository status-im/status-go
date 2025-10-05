import pytest


@pytest.mark.rpc
class TestSettings:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.setting = backend_new_profile("sender")

    def test_verify_node_config(self):
        self.setting.settings_service.node_config()
