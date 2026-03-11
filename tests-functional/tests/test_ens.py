import logging

import pytest

import resources.constants as constants

logger = logging.getLogger(__name__)

ANVIL_RPC_URL = "http://anvil:8545"
ENS_USERNAME = "testuser"
ENS_FULL_NAME = f"{ENS_USERNAME}.stateofus.eth"
CHAIN_ID = constants.ANVIL_NETWORK_ID


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
    cmd = f"/app/sync_registry.sh {registry_addr} {well_known} {ANVIL_RPC_URL} {user_namehash}"
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

    @pytest.fixture(autouse=True)
    def setup(self, backend_new_profile, foundry_client, ens_addresses, multicall3_deployer):
        self.foundry = foundry_client
        self.ens_addresses = ens_addresses
        self.rpc_client = backend_new_profile(
            name="ens_user",
            multicall_contract_address=multicall3_deployer.contract_address,
        )

    def test_ens_register_and_verify(self):
        public_key = self.rpc_client.public_key
        assert public_key, "Backend public key not available"

        price = self.rpc_client.ens_service.price(CHAIN_ID)
        assert price, "ens_price returned empty"
        logger.info(f"ENS registration price: {price}")

        registrar_addr = self.rpc_client.ens_service.get_registrar_address(CHAIN_ID)
        assert registrar_addr, "ens_getRegistrarAddress returned empty"
        assert (
            registrar_addr == self.ens_addresses["registrar"]
        ), f"Registrar mismatch: RPC={registrar_addr}, deployed={self.ens_addresses['registrar']}"

        register_ens_name(
            self.foundry,
            self.ens_addresses,
            ENS_USERNAME,
            constants.DEPLOYER_ACCOUNT.address,
            public_key,
        )
        logger.info(f"Registered {ENS_FULL_NAME} on-chain")

        # Sync registered name from deployed registry to well-known address for Go
        sync_registry_to_well_known(self.foundry, self.ens_addresses["registry"], ENS_USERNAME)

        resolved_pubkey = self.rpc_client.ens_service.public_key_of(CHAIN_ID, ENS_FULL_NAME)
        assert resolved_pubkey, "ens_publicKeyOf returned empty"

        self.rpc_client.ens_service.add(CHAIN_ID, ENS_FULL_NAME)

        usernames = self.rpc_client.ens_service.get_ens_usernames()
        assert usernames, "ens_getEnsUsernames returned empty"
        found = any(u.get("username") == ENS_FULL_NAME for u in usernames)
        assert found, f"{ENS_FULL_NAME} not found in {usernames}"
