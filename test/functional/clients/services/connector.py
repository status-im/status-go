from clients.rpc import RpcClient
from clients.services.service import Service


class ConnectorService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "connector")

    def request_accounts_accepted(self, request_id: str, account: str, chain_id: int):
        params = {
            "requestId": request_id,
            "account": account,
            "chainId": chain_id,
        }
        response = self.rpc_request("requestAccountsAccepted", [params])
        return response

    def send_transaction_accepted(self, request_id: str, tx_hash: str):
        params = {
            "requestId": request_id,
            "hash": tx_hash,
        }
        response = self.rpc_request("sendTransactionAccepted", [params])
        return response
