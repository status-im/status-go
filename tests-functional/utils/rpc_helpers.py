"""Normalize JSON-RPC results that may be null or use null list fields."""


def list_or_empty(value) -> list:
    """Return *value* if it is a list, else []."""
    return value if isinstance(value, list) else []


def _get_list(response, key: str) -> list:
    if not isinstance(response, dict):
        return []
    value = response.get(key)
    return value if isinstance(value, list) else []


def messages_list(response) -> list:
    """Extract ``messages`` from a wakuext response; null-safe for iteration."""
    return _get_list(response, "messages")


def emoji_reactions_list(response) -> list:
    """Extract ``emojiReactions`` from sendEmojiReaction RPC; null-safe for iteration."""
    return _get_list(response, "emojiReactions")
