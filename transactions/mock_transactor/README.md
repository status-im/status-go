# To generate mocks, from status-go root directory:
mockgen -source=transactions/transactor.go -destination=transactions/mock_transactor/transactor.go -package=mock_transactor
