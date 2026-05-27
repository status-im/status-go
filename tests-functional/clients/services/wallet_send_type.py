"""Wallet router send types — mirrors services/wallet/router/sendtype/send_type.go."""

from enum import IntEnum


class WalletSendType(IntEnum):
    TRANSFER = 0
    ENS_REGISTER = 1
    ENS_RELEASE = 2
    ENS_SET_PUB_KEY = 3
    STICKERS_BUY = 4
    BRIDGE = 5
    ERC721_TRANSFER = 6
    ERC1155_TRANSFER = 7
    SWAP = 8
    COMMUNITY_BURN = 9
    COMMUNITY_DEPLOY_ASSETS = 10
    COMMUNITY_DEPLOY_COLLECTIBLES = 11
    COMMUNITY_DEPLOY_OWNER_TOKEN = 12
    COMMUNITY_MINT_TOKENS = 13
    COMMUNITY_REMOTE_BURN = 14
    COMMUNITY_SET_SIGNER_PUB_KEY = 15
