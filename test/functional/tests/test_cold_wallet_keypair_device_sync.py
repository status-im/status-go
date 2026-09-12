"""A cold-wallet migration has to reach the user's other device: the receiving side is handleSyncKeypair,
driven here over real pairing and sync rather than the single backend the other cold-wallet modules use."""

import asyncio
import time

import pytest
from clients.api import ApiResponseError
from resources.constants import user_1
from steps.cold_wallet import add_seed_keypair, keystore_present
from steps.multidevice import login_paired_device, pair_second_device

SYNC_TIMEOUT = 120


async def _wait_for_keypair(device, key_uid, predicate, what, timeout=SYNC_TIMEOUT):
    """Poll the second device until the synced keypair satisfies the predicate."""
    deadline = time.monotonic() + timeout
    last = None
    while time.monotonic() < deadline:
        try:
            last = device.backend.accounts_service.get_keypair_by_key_uid(key_uid)
        except ApiResponseError as error:
            if "keypair is not found" not in str(error):
                raise
            last = None
        if last is not None and predicate(last):
            return last
        await asyncio.sleep(2)
    raise AssertionError(f"Second device never saw {what} for {key_uid} (last seen: {last})")


@pytest.mark.rpc
@pytest.mark.asyncio
class TestColdWalletKeypairDeviceSync:

    async def _paired_devices(self, async_backend_new_profile, async_backend_factory):
        primary, secondary = await asyncio.gather(
            async_backend_new_profile("cold-sync-primary"),
            async_backend_factory("cold-sync-secondary"),
        )
        secondary.backend.init_status_backend()
        await pair_second_device(primary, secondary)
        await login_paired_device(secondary, primary.backend.key_uid, primary.backend.password)
        # Don't race the second device's Waku subscriptions.
        await asyncio.to_thread(secondary.backend.wait_for_online, timeout=60)
        return primary, secondary

    async def test_a_cold_wallet_migration_reaches_the_other_device(self, async_backend_new_profile, async_backend_factory):
        primary, secondary = await self._paired_devices(async_backend_new_profile, async_backend_factory)

        key_uid = add_seed_keypair(primary.backend)["key-uid"]
        synced = await _wait_for_keypair(secondary, key_uid, lambda kp: kp.get("key-uid") == key_uid, "the imported keypair")
        assert synced.get("cold-wallet", "") == "", "Expected the keypair to arrive off any cold wallet"
        before_xpub = synced["xpub"]

        primary.backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, primary.backend.password, "status-keycard")

        arrived = await _wait_for_keypair(secondary, key_uid, lambda kp: kp.get("cold-wallet") == "status-keycard", "the cold-wallet migration")
        assert arrived["xpub"] == before_xpub, "Expected the xpub to survive the sync, since the second device cannot derive one"
        assert arrived["derived-from"] == synced["derived-from"]
        assert not any(
            keystore_present(secondary.backend, arrived).values()
        ), "Expected no keystore files on the second device either, because the key is on the card"

    async def test_migrating_back_to_the_app_reaches_the_other_device(self, async_backend_new_profile, async_backend_factory):
        primary, secondary = await self._paired_devices(async_backend_new_profile, async_backend_factory)

        key_uid = add_seed_keypair(primary.backend)["key-uid"]
        await _wait_for_keypair(secondary, key_uid, lambda kp: kp.get("key-uid") == key_uid, "the imported keypair")
        primary.backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, primary.backend.password, "status-keycard")
        await _wait_for_keypair(secondary, key_uid, lambda kp: kp.get("cold-wallet") == "status-keycard", "the cold-wallet migration")

        primary.backend.accounts_service.migrate_non_profile_cold_wallet_keypair_to_app(user_1.passphrase, primary.backend.password)

        cleared = await _wait_for_keypair(secondary, key_uid, lambda kp: kp.get("cold-wallet", "") == "", "the migration back to the app")
        assert not any(
            keystore_present(secondary.backend, cleared).values()
        ), "Expected the second device to have no keystore files, because only the first device had the seed"

    async def test_the_exact_cold_wallet_type_survives_the_sync(self, async_backend_new_profile, async_backend_factory):
        # The handler stores whatever cold_wallet string arrives; an unknown one is not reachable over
        # RPC, so this drives the nearest case: a known type the receiving device must not normalise.
        primary, secondary = await self._paired_devices(async_backend_new_profile, async_backend_factory)

        key_uid = add_seed_keypair(primary.backend)["key-uid"]
        synced = await _wait_for_keypair(secondary, key_uid, lambda kp: kp.get("key-uid") == key_uid, "the imported keypair")

        primary.backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, primary.backend.password, "ledger")

        arrived = await _wait_for_keypair(secondary, key_uid, lambda kp: kp.get("cold-wallet") == "ledger", "the ledger migration")
        assert arrived["xpub"] == synced["xpub"], "Expected the xpub to survive a ledger migration as it does a keycard one"
