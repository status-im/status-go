import os
from uuid import uuid4
import asyncio
import pytest
from src.env_vars import NUM_CONTACT_REQUESTS
from src.libs.common import delay
from src.node.status_node import StatusNode, logger
from src.steps.common import StepsCommon


def get_project_root():
    """Returns the root directory of the project."""
    return os.path.abspath(os.path.join(os.path.dirname(__file__), "../../.."))


class TestContactRequest(StepsCommon):

    @pytest.mark.asyncio
    async def test_contact_request_baseline(self, recover_network_fn=None):
        timeout_secs = 180
        reset_network_in_secs = 30
        num_contact_requests = NUM_CONTACT_REQUESTS

        project_root = get_project_root()

        nodes: list[tuple[StatusNode, StatusNode, int]] = []

        for index in range(num_contact_requests):
            # Step 1: Start status-backend and initialize application for both nodes
            first_node = StatusNode(name=f"first_node_{index}")
            second_node = StatusNode(name=f"second_node_{index}")

            data_dir_first = os.path.join(project_root, f"tests-functional/local/data{index}_first")
            data_dir_second = os.path.join(project_root, f"tests-functional/local/data{index}_second")

            delay(2)
            first_node.start(data_dir=data_dir_first)
            second_node.start(data_dir=data_dir_second)

            # Step 2: Create account and login
            account_data_first = {
                "rootDataDir": data_dir_first,
                "displayName": f"test_user_first_{index}",
                "password": f"test_password_first_{index}",
                "customizationColor": "primary"
            }
            account_data_second = {
                "rootDataDir": data_dir_second,
                "displayName": f"test_user_second_{index}",
                "password": f"test_password_second_{index}",
                "customizationColor": "primary"
            }

            first_node.create_account_and_login(account_data_first)
            second_node.create_account_and_login(account_data_second)

            delay(5)

            # Step 3: Start the Waku messenger
            first_node.start_messenger()
            second_node.start_messenger()

            # Step 4: Wait until nodes are fully started before proceeding
            first_node.wait_fully_started()
            second_node.wait_fully_started()

            nodes.append((first_node, second_node, index))

        # Step 5: Create tasks for sending contact requests
        tasks = [
            asyncio.create_task(
                self.send_and_wait_for_message((first_node, second_node), index, timeout_secs)
            )
            for first_node, second_node, index in nodes
        ]

        # Step 6: Wait for tasks with network recovery logic
        done, pending = await asyncio.wait(tasks, timeout=reset_network_in_secs)
        if pending:
            if recover_network_fn is not None:
                recover_network_fn()
                done2, _ = await asyncio.wait(pending)
                done.update(done2)
        else:
            logger.info("No pending tasks.")

        # Step 7: Collect any missing contact requests
        missing_contact_requests = []
        for task in done:
            if task.exception():
                logger.info(f"Task raised an exception: {task.exception()}")
            else:
                res = task.result()
                if res is not None:
                    missing_contact_requests.append(res)

        # Step 8: Assert if there are missing contact requests
        if missing_contact_requests:
            formatted_missing_requests = [
                f"Timestamp: {ts}, Message: {msg}, ID: {mid}" for ts, msg, mid in missing_contact_requests
            ]
            raise AssertionError(
                f"{len(missing_contact_requests)} contact requests out of {num_contact_requests} didn't reach the peer node: "
                + "\n".join(formatted_missing_requests)
            )

    async def send_and_wait_for_message(self, nodes: tuple[StatusNode, StatusNode], index: int, timeout: int = 45):
        first_node, second_node = nodes

        # Step 4: Get the public key from the first node
        first_node_pubkey = first_node.get_pubkey()

        # Prepare the contact request message
        contact_request_message = f"contact_request_{index}"
        timestamp, message_id = self.send_with_timestamp(
            second_node.send_contact_request, first_node_pubkey, contact_request_message
        )

        # Wait for the contact request message to be acknowledged
        contact_requests_successful = await first_node.wait_for_logs_async(
            [f"message received: {contact_request_message}", "AcceptContactRequest"], timeout
        )

        # Stop both nodes after message processing
        first_node.stop()
        second_node.stop()

    @pytest.mark.asyncio
    async def test_contact_request_with_latency(self):
        with self.add_latency() as recover_network_fn:
            await self.test_contact_request_baseline(recover_network_fn)

    @pytest.mark.asyncio
    async def test_contact_request_with_packet_loss(self):
        with self.add_packet_loss() as recover_network_fn:
            await self.test_contact_request_baseline(recover_network_fn)

    @pytest.mark.asyncio
    async def test_contact_request_with_low_bandwidth(self):
        with self.add_low_bandwidth() as recover_network_fn:
            await self.test_contact_request_baseline(recover_network_fn)

    def test_contact_request_with_node_pause(self, start_2_nodes):
        with self.node_pause(self.second_node):
            message = str(uuid4())
            self.first_node.send_contact_request(self.second_node_pubkey, message)
            delay(10)
        assert self.second_node.wait_for_logs([message])
