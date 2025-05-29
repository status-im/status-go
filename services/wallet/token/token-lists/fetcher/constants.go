package fetcher

// RemoteListOfTokenLists is the URL to fetch the list of token lists from. It needs to follow the schema defined below.
// #nosec G101
const RemoteListOfTokenLists = "https://prod.market.status.im/static/token-lists.json"

// sourceList is a hardcoded list of URLs to fetch token lists from (list format as below) will be used if fetching the remote list fails.
// #nosec G101
const defaultListOfTokenLists = `[
  {
    "id": "uniswap",
    "sourceUrl": "https://ipfs.io/ipns/tokens.uniswap.org",
    "schema": "https://uniswap.org/tokenlist.schema.json"
  },
  {
    "id": "aave",
    "sourceUrl": "https://raw.githubusercontent.com/bgd-labs/aave-address-book/main/tokenlist.json"
  }
]`

// #nosec G101
const listOfTokenListsSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "array",
  "items": {
    "type": "object",
    "properties": {
      "id": {
        "type": "string",
        "description": "A unique identifier for the token list source."
      },
      "sourceUrl": {
        "type": "string",
        "format": "uri",
        "description": "URL pointing to the token list source."
      },
      "schema": {
        "type": "string",
        "format": "uri",
        "description": "Optional URL pointing to the schema definition of the token list.",
        "nullable": true
      }
    },
    "required": ["id", "sourceUrl"],
    "additionalProperties": false
  }
}
`
