from clients.status_backend import StatusBackend

backend = StatusBackend(url="http://localhost:8083", data_dir="./data-3", logLevel="INFO")

backend.initialize()

backend.create_account_and_login(display_name="townhall-user-2", password="12345")
backend.wait_for_login()

backend.wakuext_service.start_messenger()
backend.wallet_service.start_wallet()

backend.settings_service.get_settings()

len(backend.wakuext_service.peers())

# vitalik.eth - 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045
# sirotin.eth - 0xF8a6c331A1140989D043Fc066B47167e5132d9A3
