"""
Light client message delivery after contact acceptance.
https://github.com/status-im/status-go/issues/7393

Two light-client nodes establish a mutual contact, then one sends a message
immediately. The test measures delivery time in two windows:

  - 0-3s:   logged as DELIVERED (fast)
  - 3-120s: logged as DELIVERED (delayed), with gap in seconds
  - >120s:  logged as NOT DELIVERED

A control variant adds a 10s delay after acceptance before sending.

Each variant runs 5 times with fresh containers to capture the
non-deterministic filter subscription timing.

Run:
    gh workflow run test-reliability.yml \\
        -f test_filter=test_light_client_contact_message_race \\
        -f waku_fleet=status.prod
"""

import asyncio
import logging
import time
from uuid import uuid4

import pytest

from clients.signals import SignalType
from steps.async_messenger import send_contact_request_and_wait

logger = logging.getLogger(__name__)


async def make_contacts_with_delay(sender, receiver, delay_after_accept: float = 0.0) -> str:
    """Establish mutual contact with an optional delay after acceptance."""
    message_id = await send_contact_request_and_wait(sender, receiver)

    accepted_signal = f"@{receiver.public_key} accepted your contact request"
    receiver.wakuext_service.accept_contact_request(message_id, sender.public_key)

    if delay_after_accept > 0:
        logger.info(f"Waiting {delay_after_accept}s after acceptance")
        await asyncio.sleep(delay_after_accept)

    try:
        await sender.wait_for_signal(
            SignalType.MESSAGES_NEW, pattern=accepted_signal, timeout=30, check_buffer=True
        )
        return message_id
    except (asyncio.TimeoutError, asyncio.CancelledError):
        logging.warning("Acceptance signal not received, falling back to RPC polling")

    deadline = time.time() + 30
    while time.time() < deadline:
        contact = sender.wakuext_service.get_contact_by_id(receiver.public_key)
        if contact and contact.get("mutual") is True:
            return message_id
        await asyncio.sleep(2)

    raise TimeoutError(f"Contact {receiver.public_key} did not become mutual on sender within timeout")


@pytest.mark.reliability
@pytest.mark.asyncio
class TestLightClientContactMessageRace:

    @pytest.fixture()
    async def sender(self, async_backend_new_profile):
        return await async_backend_new_profile("sender", waku_light_client=True, bridge_network=True)

    @pytest.fixture()
    async def receiver(self, async_backend_new_profile):
        return await async_backend_new_profile("receiver", waku_light_client=True, bridge_network=True)

    async def _send_after_contact(self, sender, receiver, run_label="", delay_after_accept=0.0):
        """Establish contact, send message, measure delivery time.

        Returns (delivered_within_3s: bool, total_seconds: float).
        """
        prefix = f"[run {run_label}] " if run_label else ""

        logger.info(f"{prefix}making contacts (delay={delay_after_accept}s)")
        await make_contacts_with_delay(sender, receiver, delay_after_accept=delay_after_accept)

        t0 = time.time()
        message_text = f"probe_{run_label}_{uuid4()}"
        sender.wakuext_service.send_one_to_one_message(receiver.public_key, message_text)
        logger.info(f"{prefix}message sent")

        # Check delivery within 3s
        delivered_fast = True
        try:
            await receiver.wait_for_signal(
                SignalType.MESSAGES_NEW, pattern=message_text, timeout=3, check_buffer=True
            )
        except (asyncio.TimeoutError, asyncio.CancelledError, TimeoutError):
            delivered_fast = False

        if delivered_fast:
            gap = time.time() - t0
            logger.info(f"{prefix}DELIVERED in {gap:.1f}s")
            return True, gap

        logger.info(f"{prefix}not delivered within 3s, waiting up to 120s")

        # Wait for MVDS retransmission
        try:
            await receiver.wait_for_signal(
                SignalType.MESSAGES_NEW, pattern=message_text, timeout=120, check_buffer=True
            )
        except (asyncio.TimeoutError, asyncio.CancelledError, TimeoutError):
            gap = time.time() - t0
            logger.error(f"{prefix}NOT DELIVERED after {gap:.1f}s")
            return False, gap

        gap = time.time() - t0
        logger.info(f"{prefix}DELIVERED after {gap:.1f}s (delayed)")
        return False, gap

    @pytest.mark.parametrize("execution_number", range(5))
    async def test_message_immediately_after_contact_accept(self, sender, receiver, execution_number):
        """Send message immediately after contact acceptance (no delay)."""
        delivered_fast, gap = await self._send_after_contact(
            sender, receiver, run_label=str(execution_number)
        )

        if not delivered_fast and gap > 120:
            pytest.fail(f"Run {execution_number}: not delivered after {gap:.1f}s")

    @pytest.mark.parametrize("execution_number", range(5))
    async def test_message_with_delay_after_contact_accept(self, sender, receiver, execution_number):
        """Send message after 10s delay following contact acceptance (control)."""
        delivered_fast, gap = await self._send_after_contact(
            sender, receiver, run_label=f"delayed_{execution_number}", delay_after_accept=10.0
        )

        if not delivered_fast and gap > 120:
            pytest.fail(f"Delayed run {execution_number}: not delivered after {gap:.1f}s")
