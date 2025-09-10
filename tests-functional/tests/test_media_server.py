import tempfile
import pytest
import requests

from clients.signals import SignalType
from clients.status_backend import StatusBackend


@pytest.mark.rpc
class TestMediaServer:
    await_signals = [
        SignalType.MEDIASERVER_STARTED.value,
        SignalType.NODE_STARTED.value,
        SignalType.NODE_READY.value,
        SignalType.NODE_LOGIN.value,
    ]

    @pytest.fixture()
    def backend(self, backend_new_profile) -> StatusBackend:
        return backend_new_profile("sender")

    def test_media_server_health(self, backend):
        signal = backend.wait_for_signal(SignalType.MEDIASERVER_STARTED.value)
        event = signal["event"]

        # Event port should match advertized port, not the actual port we're listening on
        assert event["port"] == backend.media_server_port

        media_server_url = f"https://localhost:{backend.media_server_port}"
        certificate = backend.image_server_tls_cert()

        with tempfile.NamedTemporaryFile(mode="w", suffix=".crt", delete=False) as cert_file:
            # Save the certificate to a temporary file
            cert_file.write(certificate)
            cert_file_path = cert_file.name

        # Make a health request to the media server with the loaded certificate
        response = requests.get(media_server_url + "/health", timeout=10, verify=cert_file_path)
        assert response.status_code == 200
