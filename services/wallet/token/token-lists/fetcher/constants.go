package fetcher

// TODO: update the link once we decide where to host the list of token lists.
// remoteListOfTokenLists is the URL to fetch the list of token lists from. It needs to follow the schema defined below.
// var remoteListOfTokenLists = fmt.Sprintf("https://raw.githubusercontent.com/status-im/status-go/refs/heads/release/%s/token-lists.json", version.Version())

// sourceList is a hardcoded list of URLs to fetch token lists from (list format as below) will be used if fetching the remote list fails.
// #nosec G101
const defaultListOfTokenLists = `[
  {
    "id": "coingecko",
    "sourceUrl":  "https://api.coingecko.com/api/v3/coins/list?include_platform=true"
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
