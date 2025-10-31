from clients.status_backend import StatusBackend

backend = StatusBackend(
    url="http://localhost:8080",
    data_dir="./data-1",
)

# community_id="0x02b5bdaf5a25fcfe2ee14c501fab1836b8de57f61621080c3d52073d16de0d98d6"
# Create Community
backend.wakuext_service.create_community(
    name="townhall-community",
    description="Discuss townhall demos",
)

# Join Community
response = backend.sharedurls_service.parse_shared_url(  # type: ignore
    "https://status.app/c/G3kAAMSQtb05kog3aGbr3kiaxN4th...",
)
backend.wakuext_service.request_to_join_community(
    community_id=response["community"]["communityId"],
)

# Send Message
backend.wakuext_service.send_chat_message(
    chat_id="0x02b5bdaf5a25fcfe2ee14c501fab1...",
    message="Hello from townhall demo!",
)

# Check account balances
backend.wallet_service.get_balances_at_by_chain(
    chains=[1],
    addresses=["0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"],
    tokens=[
        "0x0000000000000000000000000000000000000000",
        "0x744d70FDBE2Ba4CF95131626614a1763DF805B9E",
    ],
)
