from hashlib import shake_256
import base58

SECP256K1_PUB_CODE = 0xE701  # Multicodec code for secp256k1-pub
BASE58_CHARS = set("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")


def compress_public_key(public_key):
    if not public_key.startswith("0x"):
        public_key = "0x" + public_key
    if len(public_key) != 132:
        raise ValueError("Invalid public key")
    x = public_key[4:68]  # Extract X coordinate (first 32 bytes after prefix)
    y = public_key[68:132]  # Extract Y coordinate (last 32 bytes)
    prefix = "03" if int(y, 16) % 2 else "02"  # Add prefix 02 for even Y, 03 for odd Y
    return "0x" + prefix + x


def shake256(msg):
    h = shake_256()
    h.update(msg)
    return "0x" + h.hexdigest(64)


def read_varint(data: bytes) -> tuple[int, int]:
    """Read varint from bytes, return (value, bytes_consumed)"""
    result = 0
    shift = 0
    i = 0
    while i < len(data):
        b = data[i]
        result |= (b & 0x7F) << shift
        i += 1
        if (b & 0x80) == 0:
            break
        shift += 7
    return result, i


def varint_encode(value: int) -> bytes:
    """Encode int to varint bytes"""
    if value == 0:
        return b"\x00"
    bytes_list = []
    while value:
        byte = value & 0x7F
        value >>= 7
        if value:
            byte |= 0x80
        bytes_list.append(byte)
    return bytes(bytes_list)


def is_compressed_pub_key(pubkey: str) -> bool:
    """Check if pubkey is in multiformat zQ compressed format (len 48-50, starts with zQ3sh, base58 chars)"""
    length = len(pubkey)
    return 48 <= length <= 50 and pubkey.startswith("zQ3sh") and all(c in BASE58_CHARS for c in pubkey)


def zq_to_hex33(zq: str) -> str:
    """Convert zQ multibase multicodec compressed pubkey to 66-char hex (33 bytes compressed pubkey)"""
    payload = base58.b58decode(zq)
    code, offset = read_varint(payload)
    if code != SECP256K1_PUB_CODE:
        raise ValueError(f"Expected secp256k1-pub (0x{SECP256K1_PUB_CODE:04x}), got {code:#x}")
    pubkey_bytes = payload[offset : offset + 33]
    if len(pubkey_bytes) != 33:
        raise ValueError(f"Invalid pubkey length: {len(pubkey_bytes)} != 33")
    return pubkey_bytes.hex()


def hex33_to_zq(hex33: str) -> str:
    """Convert 66-char hex (33 bytes compressed pubkey) to zQ multibase multicodec string"""
    hex33 = hex33[2:] if hex33.startswith("0x") else hex33
    if len(hex33) != 66:
        raise ValueError(f"Invalid hex33 length: {len(hex33)} != 66")
    pubkey_bytes = bytes.fromhex(hex33)
    if len(pubkey_bytes) != 33:
        raise ValueError(f"Invalid pubkey bytes length: {len(pubkey_bytes)} != 33")
    code_bytes = varint_encode(SECP256K1_PUB_CODE)
    payload = code_bytes + pubkey_bytes
    return base58.b58encode(payload).decode("ascii")


def change_community_key_compression(community_id: str) -> str:
    """
    Emulate Nim's utl.changeCommunityKeyCompression(communityId)
    Toggles between 66-char hex compressed pubkey and zQ multiformat (~48-50 chars).
    """
    if is_compressed_pub_key(community_id):
        return zq_to_hex33(community_id)
    else:
        return hex33_to_zq(community_id)
