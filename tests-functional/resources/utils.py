def assert_response_attributes(actual, expected, keys=None):
    """
    Assert that all keys in expected (or in keys) match in actual.
    Handles both list-of-dicts and dict.
    """
    if isinstance(expected, list):
        assert isinstance(actual, list), "Expected a list for actual"
        assert len(actual) == len(expected), "Length mismatch"
        for idx, exp in enumerate(expected):
            act = actual[idx]
            check_keys = keys or exp.keys()
            for key in check_keys:
                assert act[key] == exp[key], f"Mismatch for key '{key}': {act[key]} != {exp[key]}"
    elif isinstance(expected, dict):
        assert isinstance(actual, dict), "Expected a dict for actual"
        check_keys = keys or expected.keys()
        for key in check_keys:
            assert actual[key] == expected[key], f"Mismatch for key '{key}': {actual.get(key)} != {expected[key]}"
    else:
        raise TypeError("Expected must be a list or dict")
