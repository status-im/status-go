from clients.rpc import RpcClient
from clients.services.service import Service
from tenacity import retry, stop_after_delay, wait_fixed


class WakuextService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "wakuext")

    def send_contact_request(self, contact_id: str, message: str):
        params = [{"id": contact_id, "message": message}]
        response = self.rpc_request("sendContactRequest", params)
        return response.json()

    def accept_contact_request(self, chat_id: str, retry_timeout: int = 10):
        params = [{"id": chat_id}]

        @retry(stop=stop_after_delay(retry_timeout), wait=wait_fixed(0.5), reraise=True)
        def make_request():
            response = self.rpc_request("acceptContactRequest", params)
            return response.json()

        return make_request()

    def get_contacts(self):
        response = self.rpc_request("contacts")
        return response.json()

    def send_message(self, contact_id: str, message: str):
        params = [{"id": contact_id, "message": message}]
        response = self.rpc_request("sendOneToOneMessage", params)
        return response.json()

    @retry(stop=stop_after_delay(10), wait=wait_fixed(0.5), reraise=True)
    def start_messenger(self, retry_timeout: int = 10):

        @retry(stop=stop_after_delay(retry_timeout), wait=wait_fixed(0.5), reraise=True)
        def make_request():
            response = self.rpc_request("startMessenger")
            json_response = response.json()

            if "error" in json_response:
                assert json_response["error"]["code"] == -32000
                assert json_response["error"]["message"] == "messenger already started"
                return

        make_request()

    def create_group_chat_with_members(self, pubkey_list: list, group_chat_name: str, retry_timeout: int = 10):
        params = [None, group_chat_name, pubkey_list]

        @retry(stop=stop_after_delay(retry_timeout), wait=wait_fixed(0.5), reraise=True)
        def make_request():
            response = self.rpc_request("createGroupChatWithMembers", params)
            return response.json()

        return make_request()
