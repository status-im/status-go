# To generate mocks, from status-go root directory:
mockgen -source=services/wallet/bridge/bridge.go -destination=services/wallet/bridge/mock_bridge/bridge.go -package=mock_bridge
