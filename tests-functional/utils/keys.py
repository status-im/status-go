from hashlib import shake_256


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
