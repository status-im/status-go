from time import sleep, time
from uuid import uuid4
import pytest

from steps.messenger import MessengerSteps
from clients.signals import SignalType
from resources.enums import MessageContentType


@pytest.mark.reliability
class TestLightClientRateLimiting(MessengerSteps):

    def test_light_client_rate_limiting(self):
        self.sender = self.initialize_backend(await_signals=self.await_signals, wakuV2LightClient=True)
        self.receiver = self.initialize_backend(await_signals=self.await_signals, wakuV2LightClient=True)
        sent_messages = []

        for i in range(200):
            message_text = f"test_message_{i+1}_{uuid4()}"
            response = self.sender.wakuext_service.send_message(self.receiver.public_key, message_text)
            expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
            sent_messages.append(expected_message)
            sleep(0.1)

        start_time = time()
        count = 0
        for i, expected_message in enumerate(sent_messages):
            try:
                messages_new_event = self.receiver.find_signal_containing_pattern(
                    SignalType.MESSAGES_NEW.value,
                    event_pattern=expected_message.get("id"),
                    timeout=120,
                )
                self.validate_signal_event_against_response(
                    signal_event=messages_new_event,
                    fields_to_validate={"text": "text"},
                    expected_message=expected_message,
                )
                count += 1
            except Exception:
                pass
        assert count == 200, "Not all messages were received"
        elapsed_time = time() - start_time

        assert elapsed_time >= 30, f"Message sending was too fast: {elapsed_time:.2f} seconds. Rate limiting is not applied"
        assert elapsed_time <= 90, f"Message sending took too long: {elapsed_time:.2f} seconds. Rate limiting is too high"
