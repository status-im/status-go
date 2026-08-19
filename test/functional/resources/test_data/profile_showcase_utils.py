def dummy_profile_showcase_preferences(with_collectibles: bool):
    preferences = {
        "communities": [
            {
                "communityId": "0x254254546768764565565",
                "showcaseVisibility": 3,
                "order": 0,
            },
            {
                "communityId": "0x865241434343432412343",
                "showcaseVisibility": 2,
                "order": 0,
            },
        ],
        "accounts": [
            {
                "address": "0x0000000000000000000000000033433445133423",
                "showcaseVisibility": 3,
                "order": 0,
            },
            {
                "address": "0x0000000000000000000000000032433445133424",
                "showcaseVisibility": 2,
                "order": 1,
            },
        ],
        "verifiedTokens": [
            {
                "symbol": "ETH",
                "showcaseVisibility": 3,
                "order": 1,
            },
            {
                "symbol": "DAI",
                "showcaseVisibility": 1,
                "order": 2,
            },
            {
                "symbol": "SNT",
                "showcaseVisibility": 0,
                "order": 3,
            },
        ],
        "unverifiedTokens": [
            {
                "contractAddress": "0x454525452023452",
                "chainId": 11155111,
                "showcaseVisibility": 3,
                "order": 0,
            },
            {
                "contractAddress": "0x12312323323233",
                "chainId": 1,
                "showcaseVisibility": 2,
                "order": 1,
            },
        ],
        "socialLinks": [
            {
                "text": "TwitterID",
                "url": "https://twitter.com/ethstatus",
                "showcaseVisibility": 3,
                "order": 1,
            },
            {
                "text": "TwitterID",
                "url": "https://twitter.com/StatusIMBlog",
                "showcaseVisibility": 1,
                "order": 2,
            },
            {
                "text": "GithubID",
                "url": "https://github.com/status-im",
                "showcaseVisibility": 2,
                "order": 3,
            },
        ],
    }

    if with_collectibles:
        preferences["collectibles"] = [
            {
                "contractAddress": "0x12378534257568678487683576",
                "chainId": 1,
                "tokenId": "12321389592999903",
                "showcaseVisibility": 3,
                "order": 0,
            }
        ]
    else:
        preferences["collectibles"] = []

    return preferences
