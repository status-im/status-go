import threading
import time
import traceback

from py_expression_eval import Parser

from clients.status_backend import StatusBackend
from resources.enums import MessageContentType

backend = StatusBackend(url="http://localhost:8083", data_dir="./data-1", logLevel="INFO")
backend.start_with_account("townhall-bot", "12345", "/Users/sirotin/Downloads/pepe-bot.png")
backend.generate_profile_qr_code()


def format_balances(balances: list) -> str:
    out = ""
    for token in balances:
        symbol = token["symbol"]
        balance_per_chain = token["balancesPerChain"]
        if not balance_per_chain.get("1"):
            continue
        balance_mainnet = token["balancesPerChain"]["1"]
        balance = balance_mainnet["balance"]
        out += f"- {symbol} {balance}\n"
    return out


def handle_text_message(message: dict):
    sender = message["from"]
    text = message["text"]

    if text.lower().startswith("/solve"):
        expression = text.replace("/solve", "").strip()
        try:
            result = Parser().parse(expression).evaluate({})
        except Exception as e:
            print(f"❌  Expression failed for {sender}: {expression} -> {str(e)}")
            return f"Error: {str(e)}"
        else:
            print(f"🤓️  Expression solved for {sender}: {expression} = {result}")
            return str(result)

    if text.lower().startswith("/balance"):
        address = text.replace("/balance", "").strip().lower()

        watchonly_accounts = backend.accounts_service.get_watch_only_accounts() or []
        watchonly_addresses = [a["address"].lower() for a in watchonly_accounts]

        if address.lower() in watchonly_addresses:
            print(f"🔍  Address {address} already in watch-only accounts")
        else:
            backend.accounts_service.add_watch_only_account(address, "")
            time.sleep(10)

        response = backend.wallet_service.fetch_or_get_cached_wallet_balances([address], True)
        reply = format_balances(response[address])

        print(f"💰  Balances for {sender}, address {address}: {reply}")
        return f"Balances at mainnet:\n{reply}"

    reply = text
    print(f"➡️  Echoing message from to {sender}: {text}")
    return f"Echo from Python: {reply}"


def process_message(message: dict):
    try:
        match message["contentType"]:
            case MessageContentType.CONTACT_REQUEST.value:
                print(f"🤝  Accepting contact request from {message['displayName']} (public key: {message['from']})")
                backend.wakuext_service.accept_contact_request(message["id"])
            case MessageContentType.TEXT_PLAIN.value:
                reply = handle_text_message(message)
                backend.wakuext_service.send_chat_message(message["from"], reply, responseTo=message["id"])
    except Exception as e:
        print(f"❌  Error processing message: {str(e)}")
        traceback.print_exc()


try:
    while True:
        response = backend.wait_for_messages(timeout=None)
        event = response["event"]
        if not event or "messages" not in event:
            continue
        for message in event["messages"]:
            thread = threading.Thread(target=process_message, args=(message,))
            thread.start()
except KeyboardInterrupt:
    backend.logout()
    print("Exiting...")
