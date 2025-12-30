import tempfile

import pytest
import requests

from clients.signals import SignalType


@pytest.mark.rpc
class TestMediaServer:

    def test_media_server_health(self, backend_new_profile):
        backend = backend_new_profile("sender")
        with backend.expect_signal(SignalType.MEDIASERVER_STARTED, timeout=60, start="beginning") as exp:
            pass
        event = exp.result["event"]

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

    def test_multiple_media_servers(self, backend_new_profile):
        backend_1 = backend_new_profile("backend_1")
        backend_2 = backend_new_profile("backend_2")

        with backend_1.expect_signal(SignalType.MEDIASERVER_STARTED, timeout=60, start="beginning") as exp1:
            pass
        _ = exp1.result["event"]
        with backend_2.expect_signal(SignalType.MEDIASERVER_STARTED, timeout=60, start="beginning") as exp2:
            pass
        _ = exp2.result["event"]

        with tempfile.NamedTemporaryFile(mode="w", suffix=".crt", delete=False) as cert_file:
            cert_file.write(backend_1.image_server_tls_cert())
            cert_file_path_1 = cert_file.name

        with tempfile.NamedTemporaryFile(mode="w", suffix=".crt", delete=False) as cert_file:
            cert_file.write(backend_2.image_server_tls_cert())
            cert_file_path_2 = cert_file.name

        response = requests.get(f"https://localhost:{backend_1.media_server_port}/health", timeout=10, verify=cert_file_path_1)
        assert response.status_code == 200

        response = requests.get(f"https://localhost:{backend_2.media_server_port}/health", timeout=10, verify=cert_file_path_2)
        assert response.status_code == 200
