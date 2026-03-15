import logging
import uuid

import pytest

import resources.constants as constants
from utils import wallet_utils

logger = logging.getLogger(__name__)

ANVIL_RPC_URL = "http://anvil:8545"
CHAIN_ID = constants.ANVIL_NETWORK_ID


def random_ens_username():
    return f"test{uuid.uuid4().hex[:8]}"


def cast_send(foundry, to, sig, args, private_key=constants.DEPLOYER_ACCOUNT.private_key):
    args_str = " ".join(str(a) for a in args)
    cmd = f"cast send {to} '{sig}' {args_str} --rpc-url {ANVIL_RPC_URL} --private-key {private_key}"
    result = foundry.container.exec_run(cmd)
    if result.exit_code != 0:
        raise RuntimeError(f"cast send failed: {result.output.decode().strip()}")
    return result.output.decode().strip()


def cast_call(foundry, to, sig, args=None):
    args_str = " ".join(str(a) for a in args) if args else ""
    cmd = f"cast call {to} '{sig}' {args_str} --rpc-url {ANVIL_RPC_URL}"
    result = foundry.container.exec_run(cmd)
    if result.exit_code != 0:
        raise RuntimeError(f"cast call failed: {result.output.decode().strip()}")
    return result.output.decode().strip()


def cast_calldata(foundry, sig, args):
    args_str = " ".join(str(a) for a in args)
    cmd = f"cast calldata '{sig}' {args_str}"
    result = foundry.container.exec_run(cmd)
    if result.exit_code != 0:
        raise RuntimeError(f"cast calldata failed: {result.output.decode().strip()}")
    return result.output.decode().strip()


def cast_keccak(foundry, text):
    cmd = f"cast keccak '{text}'"
    result = foundry.container.exec_run(cmd)
    if result.exit_code != 0:
        raise RuntimeError(f"cast keccak failed: {result.output.decode().strip()}")
    return result.output.decode().strip()


def extract_pubkey_coordinates(public_key):
    raw = public_key
    if raw.startswith("0x"):
        raw = raw[2:]
    if raw.startswith("04"):
        raw = raw[2:]
    x = "0x" + raw[:64]
    y = "0x" + raw[64:128]
    return x, y


def sync_registry_to_well_known(foundry, registry_addr, username):
    """Sync deployed registry storage to well-known address so Go code can read it."""
    well_known = "0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e"
    user_namehash = f"$(cast namehash '{username}.stateofus.eth')"
    cmd = f"/app/sync_ens_registry.sh {registry_addr} {well_known} {ANVIL_RPC_URL} {user_namehash}"
    result = foundry.container.exec_run(["sh", "-c", cmd])
    if result.exit_code != 0:
        raise RuntimeError(f"sync_registry failed: {result.output.decode().strip()}")


def register_ens_name(foundry, ens_addresses, username, account_address, public_key):
    token = ens_addresses["token"]
    registrar = ens_addresses["registrar"]

    label = cast_keccak(foundry, username)
    x, y = extract_pubkey_coordinates(public_key)

    extra_data = cast_calldata(
        foundry,
        "register(bytes32,address,bytes32,bytes32)",
        [label, account_address, x, y],
    )

    price_raw = cast_call(foundry, registrar, "getPrice()(uint256)")
    price = price_raw.strip().split()[0]

    cast_send(
        foundry,
        token,
        "generateTokens(address,uint256)",
        [constants.DEPLOYER_ACCOUNT.address, price],
    )

    cast_send(
        foundry,
        token,
        "approveAndCall(address,uint256,bytes)",
        [registrar, price, extra_data],
    )


@pytest.mark.rpc
@pytest.mark.ens
class TestEnsRegistration:

    @pytest.fixture()
    def backend(self, backend_new_profile, foundry_client, ens_addresses, multicall3_deployer):
        self.foundry = foundry_client
        self.ens_addresses = ens_addresses
        return backend_new_profile(
            name="ens_user",
            multicall_contract_address=multicall3_deployer.contract_address,
        )

    def test_ens_add_external_registration(self, backend):
        """Register ENS name externally (via cast), then link it to Status profile via ens_service.add."""
        username = random_ens_username()
        full_name = f"{username}.stateofus.eth"

        public_key = backend.public_key
        assert public_key, "Backend public key not available"

        price = backend.ens_service.price(CHAIN_ID)
        assert price, "ens_price returned empty"
        logger.info(f"ENS registration price: {price}")

        registrar_addr = backend.ens_service.get_registrar_address(CHAIN_ID)
        assert registrar_addr, "ens_getRegistrarAddress returned empty"
        assert (
            registrar_addr == self.ens_addresses["registrar"]
        ), f"Registrar mismatch: RPC={registrar_addr}, deployed={self.ens_addresses['registrar']}"

        register_ens_name(
            self.foundry,
            self.ens_addresses,
            username,
            constants.DEPLOYER_ACCOUNT.address,
            public_key,
        )
        logger.info(f"Registered {full_name} on-chain")

        sync_registry_to_well_known(self.foundry, self.ens_addresses["registry"], username)

        resolved_pubkey = backend.ens_service.public_key_of(CHAIN_ID, full_name)
        assert resolved_pubkey, "ens_publicKeyOf returned empty"

        backend.ens_service.add(CHAIN_ID, full_name)

        usernames = backend.ens_service.get_ens_usernames()
        assert usernames, "ens_getEnsUsernames returned empty"
        found = any(u.get("username") == full_name for u in usernames)
        assert found, f"{full_name} not found in {usernames}"


@pytest.mark.rpc
@pytest.mark.ens
class TestEnsRouterRegistration:

    @pytest.fixture()
    def backend(self, backend_recovered_profile, foundry_client, ens_addresses, multicall3_deployer):
        self.foundry = foundry_client
        self.ens_addresses = ens_addresses
        return backend_recovered_profile(
            name="ens_router_user",
            user=constants.user_1,
            multicall_contract_address=multicall3_deployer.contract_address,
        )

    def test_ens_register_via_router(self, backend):
        """Register ENS name via wallet router (production flow used by status-app)."""
        username = random_ens_username()
        full_name = f"{username}.stateofus.eth"

        public_key = backend.public_key
        assert public_key, "Backend public key not available"

        price_hex = backend.ens_service.price(CHAIN_ID)
        assert price_hex, "ens_price returned empty"
        amount_in = f"0x{price_hex}"
        logger.info(f"ENS registration price: {amount_in}")

        token_address = self.ens_addresses["token"]
        cast_send(self.foundry, token_address, "generateTokens(address,uint256)", [constants.user_1.address, int(price_hex, 16)])

        token_key = wallet_utils.get_token_key(CHAIN_ID, token_address)
        tx_result = wallet_utils.send_router_transaction(
            backend,
            uuid=str(uuid.uuid4()),
            sendType=1,  # ENSRegister
            addrFrom=constants.user_1.address,
            addrTo=constants.user_1.address,
            amountIn=amount_in,
            amountOut="0x0",
            tokenKey=token_key,
            tokenIDIsOwnerToken=False,
            toTokenKey=token_key,
            fromChainID=CHAIN_ID,
            toChainID=CHAIN_ID,
            gasFeeMode=1,
            username=username,
            publicKey=public_key,
        )
        logger.info(f"Router registration tx: {tx_result['tx_status']}")

        sync_registry_to_well_known(self.foundry, self.ens_addresses["registry"], username)

        resolved_pubkey = backend.ens_service.public_key_of(CHAIN_ID, full_name)
        assert resolved_pubkey, "ens_publicKeyOf returned empty after router registration"

        backend.ens_service.add(CHAIN_ID, full_name)

        usernames = backend.ens_service.get_ens_usernames()
        assert usernames, "ens_getEnsUsernames returned empty"
        found = any(u.get("username") == full_name for u in usernames)
        assert found, f"{full_name} not found in {usernames}"
